package pricing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/mysteriumnetwork/discovery/db"
	promV1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/rs/zerolog/log"
)

type promClient interface {
	Query(ctx context.Context, query string, ts time.Time) (model.Value, promV1.Warnings, error)
}

type NetworkLoadMultiplierCalculator struct {
	promClient        promClient
	db                *db.DB
	lock              sync.Mutex
	cachedMultipliers map[ISO3166CountryCode]float64
	stop              chan struct{}
	stopOne           sync.Once
}

func NewNetworkLoadMultiplierCalculator(promClient promClient, db *db.DB) *NetworkLoadMultiplierCalculator {
	return &NetworkLoadMultiplierCalculator{
		promClient:        promClient,
		db:                db,
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
	multis, err := nlmc.fetchMultipliersFromDB()
	if err != nil {
		return err
	}

	nlmc.setMultiplierMapCache(multis)
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
	newCache := make(map[ISO3166CountryCode]float64)
	for k := range CountryCodeToName {
		mul, err := nlmc.updateMultiplierForCountryByLoad(k)
		if err != nil {
			log.Err(err).Msgf("could not update multiplier for %s", k)
		}
		newCache[k] = mul
		log.Trace().Msgf("updated multiplier for %s to %v", k, mul)
	}
	nlmc.setMultiplierMapCache(newCache)
	log.Debug().Msgf("country update complete")
}

func (nlmc *NetworkLoadMultiplierCalculator) updateMultiplierForCountryByLoad(isoCode ISO3166CountryCode) (float64, error) {
	mul, err := nlmc.fetchMultiplierFromProm(isoCode)
	if err != nil {
		return mul, err
	}

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

func (nlmc *NetworkLoadMultiplierCalculator) fetchMultiplierFromProm(isoCode ISO3166CountryCode) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	providers, err := nlmc.getActiveProvidersForCountry(ctx, isoCode)
	if err != nil {
		return 0, err
	}

	activeSessions, err := nlmc.getSessionsForCountry(ctx, isoCode)
	if err != nil {
		return 0, err
	}

	mul := nlmc.calculateMultiplier(providers, activeSessions)
	return mul, nil
}

// calculateMultiplier calculates a coefficient for price multiplication.
// It returns a value of 0.5 < x < 2.0.
// If there are more active sessions than providers, we want to incentivise providers by increasing service price in that region.
func (nlmc *NetworkLoadMultiplierCalculator) calculateMultiplier(providers, activeSessions int64) float64 {
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

func (nlmc *NetworkLoadMultiplierCalculator) getActiveProvidersForCountry(ctx context.Context, isoCode ISO3166CountryCode) (int64, error) {
	v, _, err := nlmc.promClient.Query(
		ctx,
		fmt.Sprintf("count(sum(proposal_event{country='%s'}[1h]) by (provider_id))", isoCode),
		time.Now(),
	)
	if err != nil {
		return 0, fmt.Errorf("could not fetch proposals for country %s", isoCode)
	}

	casted := v.(model.Vector)
	if len(casted) == 0 {
		return 0, nil
	}

	return int64(casted[0].Value), nil
}

func (nlmc *NetworkLoadMultiplierCalculator) getSessionsForCountry(ctx context.Context, isoCode ISO3166CountryCode) (int64, error) {
	v, _, err := nlmc.promClient.Query(
		ctx,
		fmt.Sprintf("count(count(session_data{provider_country='%s'}[1h]) by (session_id))", isoCode),
		time.Now(),
	)
	if err != nil {
		return 0, fmt.Errorf("could not fetch sessions for country %s", isoCode)
	}

	casted := v.(model.Vector)
	if len(casted) == 0 {
		return 0, nil
	}

	return int64(casted[0].Value), nil
}
