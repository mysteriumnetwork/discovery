package pricing

import (
	"context"
	"sync"
	"time"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/mysteriumnetwork/discovery/db"
	"github.com/mysteriumnetwork/discovery/quality/oracleapi"
	"github.com/rs/zerolog/log"
)

type qualityAPI interface {
	NetworkLoad() (oracleapi.NetworkLoadByCountry, error)
}

type NetworkLoadMultiplierCalculator struct {
	networkLoadGetter qualityAPI
	db                *db.DB
	lock              sync.Mutex
	cachedMultipliers map[ISO3166CountryCode]float64
	stop              chan struct{}
	stopOne           sync.Once
	disableUpdate     bool
}

func NewNetworkLoadMultiplierCalculator(networkLoadGetter qualityAPI, db *db.DB, disableUpdate bool) *NetworkLoadMultiplierCalculator {
	return &NetworkLoadMultiplierCalculator{
		networkLoadGetter: networkLoadGetter,
		db:                db,
		stop:              make(chan struct{}),
		cachedMultipliers: make(map[ISO3166CountryCode]float64),
		disableUpdate:     disableUpdate,
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
	multis, err := nlmc.fetchMultipliersFromDB()
	if err != nil {
		return err
	}

	nlmc.setMultiplierMapCache(multis)
	log.Info().Msgf("NetworkLoadMultiplierCalculator initial map loaded")

	go func() {
		if nlmc.disableUpdate {
			log.Info().Msgf("NetworkLoadMultiplierCalculator update disabled")
			return
		}

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

type multiplierDB struct {
	Multiplier  float64 `json:"multiplier"`
	CountryCode string  `json:"country_code"`
}

func (nlmc *NetworkLoadMultiplierCalculator) fetchMultipliersFromDB() (map[ISO3166CountryCode]float64, error) {
	conn, err := nlmc.db.Connection()
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	rows, err := conn.Query(ctx, "SELECT multiplier, country_code FROM country_price_multipliers")
	if err != nil {
		return nil, err
	}
	results := make(map[ISO3166CountryCode]float64)
	for rows.Next() {
		var res = multiplierDB{}
		if err := pgxscan.ScanRow(&res, rows); err != nil {
			return nil, err
		}
		code := ISO3166CountryCode(res.CountryCode)
		if code.Validate() != nil {
			log.Error().Msgf("invalid country code %s in db", code)
			continue
		}
		results[code] = res.Multiplier
	}

	return results, nil
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
	return mul, nlmc.storeMultiplier(isoCode, mul)
}

func (nlmc *NetworkLoadMultiplierCalculator) storeMultiplier(isoCode ISO3166CountryCode, multiplier float64) error {
	conn, err := nlmc.db.Connection()
	if err != nil {
		return err
	}
	defer conn.Release()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	query := `INSERT INTO country_price_multipliers(
		country_code, multiplier, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (country_code) DO UPDATE
			SET multiplier = $2,
				updated_at = now();`
	_, err = conn.Exec(ctx, query, isoCode.String(), multiplier)
	return err
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
