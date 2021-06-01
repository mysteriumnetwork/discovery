package pricing

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/fln/pprotect"

	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/rs/zerolog/log"
)

type Bound struct {
	Min, Max float64
}

type FiatPriceAPI interface {
	MystUSD() float64
}

// NetworkLoadByCountryProvider calculates the price multiplier according to the load of the network for the given country.
type NetworkLoadByCountryProvider interface {
	GetMultiplier(isoCode ISO3166CountryCode) float64
}

type Pricer struct {
	priceAPI              FiatPriceAPI
	priceLifetime         time.Duration
	mystBound             Bound
	loadByCountryProvider NetworkLoadByCountryProvider

	lock        sync.Mutex
	lp          LatestPrices
	cfgProvider ConfigProvider
}

func NewPricer(
	cfgProvider ConfigProvider,
	priceAPI FiatPriceAPI,
	priceLifetime time.Duration,
	sensibleMystBound Bound,
	loadByCountryProvider NetworkLoadByCountryProvider,
) (*Pricer, error) {
	pricer := &Pricer{
		cfgProvider:           cfgProvider,
		priceAPI:              priceAPI,
		priceLifetime:         priceLifetime,
		mystBound:             sensibleMystBound,
		loadByCountryProvider: loadByCountryProvider,
	}

	go schedulePriceUpdate(priceLifetime, pricer)
	if err := pricer.threadSafePriceUpdate(); err != nil {
		return nil, err
	}
	return pricer, nil
}

func schedulePriceUpdate(priceLifetime time.Duration, pricer *Pricer) {
	for {
		select {
		case <-time.After(priceLifetime):
			pprotect.CallLoop(func() {
				err := pricer.threadSafePriceUpdate()
				if err != nil {
					log.Err(err).Msg("failed to update prices")
				}
			}, time.Second, func(val interface{}, stack []byte) {
				log.Warn().Msg("panic on scheduled price update: " + fmt.Sprint(val))
			})
		}
	}
}

func (p *Pricer) threadSafePriceUpdate() error {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.updatePrices()
}

func (p *Pricer) GetPrices() LatestPrices {
	p.lock.Lock()
	defer p.lock.Unlock()

	if !p.lp.isValid() {
		err := p.updatePrices()
		if err != nil {
			log.Err(err).Msg("could not update prices")
		}
	}

	return p.lp
}

func (p *Pricer) updatePrices() error {
	mystUSD, err := p.fetchMystPrice()
	if err != nil {
		return err
	}

	cfg, err := p.cfgProvider.Get()
	if err != nil {
		return err
	}

	p.lp = p.generateNewLatestPrice(mystUSD, cfg)
	log.Info().Msg("prices updated")

	return nil
}

func (p *Pricer) generateNewLatestPrice(mystUSD float64, cfg Config) LatestPrices {
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

func (p *Pricer) generateNewDefaults(mystUSD float64, cfg Config) *PriceHistory {
	ph := &PriceHistory{
		Current: &PriceByType{
			Residential: &Price{
				PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.PricePerHour, 1),
				PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.PricePerHour, 1),
				PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.PricePerGiB, 1),
				PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.PricePerGiB, 1),
			},
			Other: &Price{
				PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Other.PricePerHour, 1),
				PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.PricePerHour, 1),
				PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Other.PricePerGiB, 1),
				PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.PricePerGiB, 1),
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

func (p *Pricer) generateNewPerCountry(mystUSD float64, cfg Config) map[string]*PriceHistory {
	countries := make(map[string]*PriceHistory)
	for countryCode := range CountryCodeToName {
		mod, ok := cfg.CountryModifiers[ISO3166CountryCode(countryCode)]
		if !ok {
			mod = Modifier{
				Residential: 1,
				Other:       1,
			}
		}

		loadModifier := p.loadByCountryProvider.GetMultiplier(countryCode)
		mod.Other *= loadModifier
		mod.Residential *= loadModifier

		ph := &PriceHistory{
			Current: &PriceByType{
				Residential: &Price{
					PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.PricePerHour, mod.Residential),
					PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.PricePerHour, mod.Residential),
					PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.PricePerGiB, mod.Residential),
					PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.PricePerGiB, mod.Residential),
				},
				Other: &Price{
					PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Other.PricePerHour, mod.Other),
					PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.PricePerHour, mod.Residential),
					PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Other.PricePerGiB, mod.Other),
					PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.PricePerGiB, mod.Residential),
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

func (p *Pricer) fetchMystPrice() (float64, error) {
	mystUSD := p.priceAPI.MystUSD()
	if err := p.withinBounds(mystUSD); err != nil {
		return 0, err
	}

	return mystUSD, nil
}

// withinBounds used to filter out any possible nonsense that the external pricing services might return.
func (p *Pricer) withinBounds(price float64) error {
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
	Residential *Price `json:"residential"`
	Other       *Price `json:"other"`
}

type Price struct {
	PricePerHour              *big.Int `json:"price_per_hour" swaggertype:"integer"`
	PricePerHourHumanReadable float64  `json:"price_per_hour_human_readable" swaggertype:"number"`
	PricePerGiB               *big.Int `json:"price_per_gib" swaggertype:"integer"`
	PricePerGiBHumanReadable  float64  `json:"price_per_gib_human_readable" swaggertype:"number"`
}
