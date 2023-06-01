package pricing

import (
	"errors"
	"sync"
	"time"

	"github.com/mysteriumnetwork/payments/exchange"
	"github.com/rs/zerolog/log"
)

type Market struct {
	lock           sync.Mutex
	stopOnce       sync.Once
	stop           chan (struct{})
	apis           []ExternalPriceAPI
	latestPrice    float64
	updateInterval time.Duration
}

func NewMarket(apis []ExternalPriceAPI, updateInterval time.Duration) *Market {
	return &Market{
		apis:           apis,
		stop:           make(chan struct{}),
		updateInterval: updateInterval,
	}
}

type ExternalPriceAPI interface {
	GetRateCacheWithFallback(coins []exchange.Coin, vsCurrencies []exchange.Currency) (exchange.PriceResponse, error)
}

func (m *Market) MystUSD() float64 {
	return m.getPrice()
}

func (m *Market) Start() error {
	if len(m.apis) == 0 {
		return errors.New("no price api providers provided")
	}

	pr, err := m.fetchPricing()
	if err != nil {
		return err
	}
	m.setPrice(pr)

	go m.periodicUpdate()
	return nil
}

func (m *Market) getPrice() float64 {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.latestPrice
}

func (m *Market) setPrice(in float64) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.latestPrice = in
}

func (m *Market) fetchPricing() (float64, error) {
	for _, v := range m.apis {
		resp, err := v.GetRateCacheWithFallback([]exchange.Coin{exchange.CoinMYST}, []exchange.Currency{exchange.CurrencyUSD})
		if err != nil {
			log.Error().Err(err).Msg("could not load pricing info")
			continue
		}

		price, ok := resp.GetRateInUSD(exchange.CoinMYST)
		if !ok {
			log.Warn().Msg("no price info for MYST found in response")
			continue
		}

		return price, nil
	}

	return 0, errors.New("could not load price info")
}

func (m *Market) periodicUpdate() {
	for {
		select {
		case <-m.stop:
			return
		case <-time.After(m.updateInterval):
			res, err := m.fetchPricing()
			if err == nil {
				m.setPrice(res)
			}
		}
	}
}

func (m *Market) Stop() {
	m.stopOnce.Do(func() {
		close(m.stop)
	})
}
