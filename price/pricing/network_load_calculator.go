package pricing

import (
	"sync"
	"time"

	"github.com/mysteriumnetwork/discovery/quality/oracleapi"
	"github.com/rs/zerolog/log"
)

type qualityAPI interface {
	NetworkLoad() (oracleapi.NetworkLoadByCountry, error)
}

type NetworkLoadMultiplierCalculator struct {
	networkLoadGetter qualityAPI
	lock              sync.Mutex
	cachedMultipliers map[ISO3166CountryCode]float64
	stop              chan struct{}
	stopOne           sync.Once
}

func NewNetworkLoadMultiplierCalculator(networkLoadGetter qualityAPI) *NetworkLoadMultiplierCalculator {
	return &NetworkLoadMultiplierCalculator{
		networkLoadGetter: networkLoadGetter,
		stop:              make(chan struct{}),
		cachedMultipliers: make(map[ISO3166CountryCode]float64),
	}
}

func (nlmc *NetworkLoadMultiplierCalculator) GetMultiplier(isoCode ISO3166CountryCode) float64 {
	fetched, ok := nlmc.getCachedMulitplier(isoCode)
	if !ok {
		log.Error().Msgf("could not fetch multiplier for country %s", isoCode)
		return 1
	}
	return fetched
}

func (nlmc *NetworkLoadMultiplierCalculator) setMultiplierMapCache(new map[ISO3166CountryCode]float64) {
	nlmc.lock.Lock()
	defer nlmc.lock.Unlock()

	nlmc.cachedMultipliers = new
}

func (nlmc *NetworkLoadMultiplierCalculator) getCachedMulitplier(isoCode ISO3166CountryCode) (float64, bool) {
	nlmc.lock.Lock()
	defer nlmc.lock.Unlock()

	v, ok := nlmc.cachedMultipliers[isoCode]
	return v, ok
}

func (nlmc *NetworkLoadMultiplierCalculator) Start() error {
	log.Info().Msgf("NetworkLoadMultiplierCalculator loading initial country map")
	nlmc.updateCountries()
	log.Info().Msgf("NetworkLoadMultiplierCalculator initial map loaded")

	go func() {
		log.Info().Msgf("NetworkLoadMultiplierCalculator started")
		for {
			select {
			case <-nlmc.stop:
				log.Info().Msgf("NetworkLoadMultiplierCalculator stopped")
				return
			case <-time.After(time.Minute * 30):
				nlmc.updateCountries()
			}
		}
	}()

	return nil
}

func (nlmc *NetworkLoadMultiplierCalculator) Stop() {
	nlmc.stopOne.Do(func() { close(nlmc.stop) })
}

func (nlmc *NetworkLoadMultiplierCalculator) updateCountries() {
	log.Debug().Msgf("country update started")

	load, err := nlmc.networkLoadGetter.NetworkLoad()
	if err != nil {
		log.Err(err).Msgf("could not get network load from quality oracle")
		return
	}

	newCache := make(map[ISO3166CountryCode]float64)
	for k := range CountryCodeToName {
		mul, err := nlmc.updateMultiplierForCountryByLoad(k, load)
		if err != nil {
			log.Err(err).Msgf("could not update multiplier for %s, will default to 1", k)
		}
		newCache[k] = mul
		log.Trace().Msgf("updated multiplier for %s to %v", k, mul)
	}
	nlmc.setMultiplierMapCache(newCache)
	log.Debug().Msgf("country update complete")
}

func (nlmc *NetworkLoadMultiplierCalculator) updateMultiplierForCountryByLoad(isoCode ISO3166CountryCode, loadMap oracleapi.NetworkLoadByCountry) (float64, error) {
	v, ok := loadMap[isoCode.String()]
	if !ok {
		return 1, nil
	}

	mul := nlmc.calculateMultiplier(v.Providers, v.Sessions)
	return mul, nil
}

// calculateMultiplier calculates a coefficient for price multiplication.
// It returns a value of 0.5 < x < 2.0.
// If there are more active sessions than providers, we want to incentivise providers by increasing service price in that region.
func (nlmc *NetworkLoadMultiplierCalculator) calculateMultiplier(providers, activeSessions uint64) float64 {
	if providers == 0 {
		return 1
	}

	coeff := float64(activeSessions) / float64(providers)
	if coeff < 0.5 {
		return 0.5
	}
	if coeff > 2 {
		return 2.0
	}

	return coeff
}
