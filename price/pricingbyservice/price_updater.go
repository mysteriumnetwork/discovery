package pricingbyservice

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/fln/pprotect"
	"github.com/go-redis/redis/v8"

	"github.com/mysteriumnetwork/discovery/metrics"
	"github.com/mysteriumnetwork/discovery/price/pricing"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/rs/zerolog/log"
)

const PriceRedisKey = "DISCOVERY_CURRENT_PRICE_BY_SERVICE"

type Bound struct {
	Min, Max float64
}

type FiatPriceAPI interface {
	MystUSD() float64
}

type PriceUpdater struct {
	priceAPI      FiatPriceAPI
	priceLifetime time.Duration
	mystBound     Bound
	db            *redis.Client

	lock        sync.Mutex
	lp          LatestPrices
	cfgProvider ConfigProvider

	stop chan struct{}
	once sync.Once
}

func NewPricer(
	cfgProvider ConfigProvider,
	priceAPI FiatPriceAPI,
	priceLifetime time.Duration,
	sensibleMystBound Bound,
	db *redis.Client,
) (*PriceUpdater, error) {
	pricer := &PriceUpdater{
		cfgProvider:   cfgProvider,
		priceAPI:      priceAPI,
		priceLifetime: priceLifetime,
		mystBound:     sensibleMystBound,
		stop:          make(chan struct{}),
		db:            db,
	}

	go pricer.schedulePriceUpdate(priceLifetime)
	if err := pricer.threadSafePriceUpdate(); err != nil {
		return nil, err
	}
	return pricer, nil
}

func (p *PriceUpdater) schedulePriceUpdate(priceLifetime time.Duration) {
	log.Info().Msg("price update be service started")
	for {
		select {
		case <-p.stop:
			log.Info().Msg("price update by service stopped")
			return
		case <-time.After(priceLifetime):
			pprotect.CallLoop(func() {
				err := p.threadSafePriceUpdate()
				if err != nil {
					log.Err(err).Msg("failed to update prices by service")
				}
			}, time.Second, func(val interface{}, stack []byte) {
				log.Warn().Msg("panic on scheduled price update by service: " + fmt.Sprint(val))
			})
		}
	}
}

func (p *PriceUpdater) threadSafePriceUpdate() error {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.updatePrices()
}

func (p *PriceUpdater) updatePrices() error {
	mystUSD, err := p.fetchMystPrice()
	if err != nil {
		return err
	}

	cfg, err := p.cfgProvider.Get()
	if err != nil {
		return err
	}

	p.lp = p.generateNewLatestPrice(mystUSD, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	marshalled, err := json.Marshal(p.lp)
	if err != nil {
		return err
	}

	err = p.db.Set(ctx, PriceRedisKey, string(marshalled), 0).Err()
	if err != nil {
		return err
	}

	p.submitMetrics()

	log.Info().Msgf("price update complete by service")
	return nil
}

func (p *PriceUpdater) submitMetrics() {
	p.submitPriceMetric("DEFAULTS", p.lp.Defaults.Current)

	for k, v := range p.lp.PerCountry {
		p.submitPriceMetric(k, v.Current)
	}
}

func (p *PriceUpdater) submitPriceMetric(country string, price *PriceByType) {
	metrics.CurrentPriceByCountry.WithLabelValues(country, "other", "wireguard", "per_gib").Set(price.Other.Wireguard.PricePerGiBHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "other", "wireguard", "per_hour").Set(price.Other.Wireguard.PricePerHourHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "residential", "wireguard", "per_gib").Set(price.Residential.Wireguard.PricePerGiBHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "residential", "wireguard", "per_hour").Set(price.Residential.Wireguard.PricePerHourHumanReadable)

	metrics.CurrentPriceByCountry.WithLabelValues(country, "other", "scraping", "per_gib").Set(price.Other.Scraping.PricePerGiBHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "other", "scraping", "per_hour").Set(price.Other.Scraping.PricePerHourHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "residential", "scraping", "per_gib").Set(price.Residential.Scraping.PricePerGiBHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "residential", "scraping", "per_hour").Set(price.Residential.Scraping.PricePerHourHumanReadable)

	metrics.CurrentPriceByCountry.WithLabelValues(country, "other", "data_transfer", "per_gib").Set(price.Other.DataTransfer.PricePerGiBHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "other", "data_transfer", "per_hour").Set(price.Other.DataTransfer.PricePerHourHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "residential", "data_transfer", "per_gib").Set(price.Residential.DataTransfer.PricePerGiBHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "residential", "data_transfer", "per_hour").Set(price.Residential.DataTransfer.PricePerHourHumanReadable)
}

func (p *PriceUpdater) Stop() {
	p.once.Do(func() { close(p.stop) })
}

func (p *PriceUpdater) generateNewLatestPrice(mystUSD float64, cfg Config) LatestPrices {
	tm := time.Now().UTC()

	newLP := LatestPrices{
		Defaults:          p.generateNewDefaults(mystUSD, cfg),
		PerCountry:        p.generateNewPerCountry(mystUSD, cfg),
		CurrentValidUntil: tm.Add(p.priceLifetime),
	}

	if !p.lp.isInitialized() {
		newLP.Defaults.Previous = newLP.Defaults.Current
		newLP.PreviousValidUntil = tm.Add(-p.priceLifetime)
	} else {
		newLP.Defaults.Previous = p.lp.Defaults.Current
		newLP.PreviousValidUntil = p.lp.CurrentValidUntil
	}
	return newLP
}

func (p *PriceUpdater) generateNewDefaults(mystUSD float64, cfg Config) *PriceHistory {
	ph := &PriceHistory{
		Current: &PriceByType{
			Residential: &PriceByServiceType{
				Wireguard: &Price{
					PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.Wireguard.PricePerHour, 1),
					PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.Wireguard.PricePerHour, 1),
					PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.Wireguard.PricePerGiB, 1),
					PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.Wireguard.PricePerGiB, 1),
				},
				Scraping: &Price{
					PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.Scraping.PricePerHour, 1),
					PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.Scraping.PricePerHour, 1),
					PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.Scraping.PricePerGiB, 1),
					PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.Scraping.PricePerGiB, 1),
				},
				DataTransfer: &Price{
					PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.DataTransfer.PricePerHour, 1),
					PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.DataTransfer.PricePerHour, 1),
					PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.DataTransfer.PricePerGiB, 1),
					PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.DataTransfer.PricePerGiB, 1),
				},
			},
			Other: &PriceByServiceType{
				Wireguard: &Price{
					PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Other.Wireguard.PricePerHour, 1),
					PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.Wireguard.PricePerHour, 1),
					PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Other.Wireguard.PricePerGiB, 1),
					PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.Wireguard.PricePerGiB, 1),
				},
				Scraping: &Price{
					PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Other.Scraping.PricePerHour, 1),
					PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.Scraping.PricePerHour, 1),
					PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Other.Scraping.PricePerGiB, 1),
					PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.Scraping.PricePerGiB, 1),
				},
				DataTransfer: &Price{
					PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Other.DataTransfer.PricePerHour, 1),
					PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.DataTransfer.PricePerHour, 1),
					PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Other.DataTransfer.PricePerGiB, 1),
					PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.DataTransfer.PricePerGiB, 1),
				},
			},
		},
	}
	if !p.lp.isInitialized() {
		ph.Previous = ph.Current
	} else {
		ph.Previous = p.lp.Defaults.Current
	}
	return ph
}

func (p *PriceUpdater) generateNewPerCountry(mystUSD float64, cfg Config) map[string]*PriceHistory {
	countries := make(map[string]*PriceHistory)
	for countryCode := range pricing.CountryCodeToName {
		mod, ok := cfg.CountryModifiers[pricing.ISO3166CountryCode(countryCode)]
		if !ok {
			mod = Modifier{
				Residential: 1,
				Other:       1,
			}
		}

		ph := &PriceHistory{
			Current: &PriceByType{
				Residential: &PriceByServiceType{
					Wireguard: &Price{
						PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.Wireguard.PricePerHour, mod.Residential),
						PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.Wireguard.PricePerHour, mod.Residential),
						PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.Wireguard.PricePerGiB, mod.Residential),
						PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.Wireguard.PricePerGiB, mod.Residential),
					},
					Scraping: &Price{
						PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.Scraping.PricePerHour, mod.Residential),
						PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.Scraping.PricePerHour, mod.Residential),
						PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.Scraping.PricePerGiB, mod.Residential),
						PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.Scraping.PricePerGiB, mod.Residential),
					},
					DataTransfer: &Price{
						PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.DataTransfer.PricePerHour, mod.Residential),
						PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.DataTransfer.PricePerHour, mod.Residential),
						PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.DataTransfer.PricePerGiB, mod.Residential),
						PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.DataTransfer.PricePerGiB, mod.Residential),
					},
				},
				Other: &PriceByServiceType{
					Wireguard: &Price{
						PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Other.Wireguard.PricePerHour, mod.Other),
						PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.Wireguard.PricePerHour, mod.Other),
						PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Other.Wireguard.PricePerGiB, mod.Other),
						PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.Wireguard.PricePerGiB, mod.Other),
					},
					Scraping: &Price{
						PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Other.Scraping.PricePerHour, mod.Other),
						PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.Scraping.PricePerHour, mod.Other),
						PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Other.Scraping.PricePerGiB, mod.Other),
						PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.Scraping.PricePerGiB, mod.Other),
					},
					DataTransfer: &Price{
						PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Other.DataTransfer.PricePerHour, mod.Other),
						PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.DataTransfer.PricePerHour, mod.Other),
						PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Other.DataTransfer.PricePerGiB, mod.Other),
						PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.DataTransfer.PricePerGiB, mod.Other),
					},
				},
			},
		}

		// if current exists in previous lp, take it, otherwise set it to current
		if p.lp.isInitialized() {
			older, ok := p.lp.PerCountry[string(countryCode)]
			if ok {
				ph.Previous = older.Current
			} else {
				ph.Previous = ph.Current
			}
		} else {
			ph.Previous = ph.Current
		}

		countries[string(countryCode)] = ph
	}
	return countries
}

// Take note that this is not 100% correct as we're rounding a bit due to accuracy issues with floats.
// This, however, is not important here as the accuracy will be more than good enough to a few zeroes after the dot.
func calculatePriceMYST(mystUSD, priceUSD, multiplier float64) *big.Int {
	return crypto.FloatToBigMyst((priceUSD / mystUSD) * multiplier)
}

func calculatePriceMystFloat(mystUSD, priceUSD, multiplier float64) float64 {
	return crypto.BigMystToFloat(calculatePriceMYST(mystUSD, priceUSD, multiplier))
}

func (p *PriceUpdater) fetchMystPrice() (float64, error) {
	mystUSD := p.priceAPI.MystUSD()
	if err := p.withinBounds(mystUSD); err != nil {
		return 0, err
	}

	return mystUSD, nil
}

// withinBounds used to filter out any possible nonsense that the external pricing services might return.
func (p *PriceUpdater) withinBounds(price float64) error {
	if price > p.mystBound.Max || price < p.mystBound.Min {
		return fmt.Errorf("myst exceeds sensible bounds: %.6f < %.6f(current price) < %.6f", p.mystBound.Min, price, p.mystBound.Max)
	}
	return nil
}

// LatestPrices holds two sets of prices. The Previous should be used in case
// a race condition between obtaining prices by Consumer and Provider
// upon agreement
type LatestPrices struct {
	Defaults           *PriceHistory            `json:"defaults"`
	PerCountry         map[string]*PriceHistory `json:"per_country"`
	CurrentValidUntil  time.Time                `json:"current_valid_until"`
	PreviousValidUntil time.Time                `json:"previous_valid_until"`
	CurrentServerTime  time.Time                `json:"current_server_time"`
}

func (lp LatestPrices) WithCurrentTime() LatestPrices {
	lp.CurrentServerTime = time.Now().UTC()
	return lp
}

func (lp *LatestPrices) isInitialized() bool {
	return lp.Defaults != nil
}

func (lp *LatestPrices) isValid() bool {
	return lp.CurrentValidUntil.UTC().After(time.Now().UTC())
}

type PriceHistory struct {
	Current  *PriceByType `json:"current"`
	Previous *PriceByType `json:"previous"`
}

type PriceByType struct {
	Residential *PriceByServiceType `json:"residential"`
	Other       *PriceByServiceType `json:"other"`
}

type PriceByServiceType struct {
	Wireguard    *Price `json:"wireguard"`
	Scraping     *Price `json:"scraping"`
	DataTransfer *Price `json:"data_transfer"`
}

type Price struct {
	PricePerHour              *big.Int `json:"price_per_hour" swaggertype:"integer"`
	PricePerHourHumanReadable float64  `json:"price_per_hour_human_readable" swaggertype:"number"`
	PricePerGiB               *big.Int `json:"price_per_gib" swaggertype:"integer"`
	PricePerGiBHumanReadable  float64  `json:"price_per_gib_human_readable" swaggertype:"number"`
}
