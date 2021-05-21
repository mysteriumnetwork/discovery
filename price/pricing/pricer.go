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
	Min float64
	Max float64
}

type PriceAPI interface {
	MystUSD() (float64, error)
}

type Pricer struct {
	cfg           Config
	priceAPI      PriceAPI
	priceLifetime time.Duration
	mystBound     Bound

	lock sync.Mutex
	lp   LatestPrices
}

func NewPricer(provider ConfigProvider, priceAPI PriceAPI, priceLifetime time.Duration, sensibleMystBound Bound) (*Pricer, error) {
	cfg, err := provider.Get()
	if err != nil {
		return nil, err
	}
	pricer := &Pricer{
		cfg:           cfg,
		priceAPI:      priceAPI,
		priceLifetime: priceLifetime,
		mystBound:     sensibleMystBound,
	}
	pprotect.CallLoop(func() {
		err := pricer.updatePrices()
		log.Err(err).Msg("Failed to update prices")
	}, priceLifetime)
	if err := pricer.init(); err != nil {
		return nil, err
	}
	return pricer, nil
}

func (p *Pricer) init() error {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.updatePrices()
}

func (p *Pricer) UpdateConfig(cfg Config) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.cfg = cfg
}

func (p *Pricer) GetPrices() LatestPrices {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.lp.CurrentValidUntil.UTC().Before(time.Now().UTC()) {
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

	log.Info().Msg("prices updated")
	p.lp = p.generateNewLatestPrice(mystUSD)
	return nil
}

func (p *Pricer) generateNewLatestPrice(mystUSD float64) LatestPrices {
	tm := time.Now().UTC()

	newLP := LatestPrices{
		Defaults:          p.generateNewDefaults(mystUSD),
		PerCountry:        p.generateNewPerCountry(mystUSD),
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

func (p *Pricer) generateNewDefaults(mystUSD float64) *PriceHistory {
	ph := &PriceHistory{
		Current: &PriceByType{
			Residential: &Price{
				PricePerHour: calculatePriceMYST(mystUSD, p.cfg.BasePrices.Residential.PricePerHour, 1),
				PricePerGiB:  calculatePriceMYST(mystUSD, p.cfg.BasePrices.Residential.PricePerGiB, 1),
			},
			Other: &Price{
				PricePerHour: calculatePriceMYST(mystUSD, p.cfg.BasePrices.Other.PricePerHour, 1),
				PricePerGiB:  calculatePriceMYST(mystUSD, p.cfg.BasePrices.Other.PricePerGiB, 1),
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

func (p *Pricer) generateNewPerCountry(mystUSD float64) map[string]*PriceHistory {
	countries := make(map[string]*PriceHistory)
	for k, v := range p.cfg.CountryModifiers {
		ph := &PriceHistory{
			Current: &PriceByType{
				Residential: &Price{
					PricePerHour: calculatePriceMYST(mystUSD, p.cfg.BasePrices.Residential.PricePerHour, v.Residential),
					PricePerGiB:  calculatePriceMYST(mystUSD, p.cfg.BasePrices.Residential.PricePerGiB, v.Residential),
				},
				Other: &Price{
					PricePerHour: calculatePriceMYST(mystUSD, p.cfg.BasePrices.Other.PricePerHour, v.Other),
					PricePerGiB:  calculatePriceMYST(mystUSD, p.cfg.BasePrices.Other.PricePerGiB, v.Other),
				},
			},
		}

		// if current exists in previous lp, take it, otherwise set it to current
		if p.lp.isInitialized() {
			older, ok := p.lp.PerCountry[string(k)]
			if ok {
				ph.Previous = older.Current
			} else {
				ph.Previous = ph.Current
			}
		} else {
			ph.Previous = ph.Current
		}

		countries[string(k)] = ph
	}
	return countries
}

// Take note that this is not 100% correct as we're rounding a bit due to accuracy issues with floats.
// This, however, is not important here as the accuracy will be more than good enough to a few zeroes after the dot.
func calculatePriceMYST(mystUSD, priceUSD, multiplier float64) *big.Int {
	return crypto.FloatToBigMyst((priceUSD / mystUSD) * multiplier)
}

func (p *Pricer) fetchMystPrice() (float64, error) {
	mystUSD, err := p.priceAPI.MystUSD()
	if err != nil {
		return 0, err
	}

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

type PriceHistory struct {
	Current  *PriceByType `json:"current"`
	Previous *PriceByType `json:"previous"`
}

type PriceByType struct {
	Residential *Price `json:"residential"`
	Other       *Price `json:"other"`
}

type Price struct {
	PricePerHour *big.Int `json:"price_per_hour" swaggertype:"integer"`
	PricePerGiB  *big.Int `json:"price_per_gib" swaggertype:"integer"`
}
