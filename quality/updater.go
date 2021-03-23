package quality

import (
	"github.com/rs/zerolog/log"
	"time"
)

type Keeper struct {
	updateCycle          time.Duration
	qualityFetchDebounce time.Duration
	qualityOracleURL     string
	countryProvider      QualityRepository
	oracleAPI            *OracleAPI
}

type QualityRepository interface {
	Countries() ([]string, error)
	StoreQuality(id, serviceType, forCountry string, quality float32) error
}

type KeeperConfig struct {
	UpdateCycle          time.Duration
	QualityFetchDebounce time.Duration
}

func NewKeeper(
	qualityOracleURL string,
	countryProvider QualityRepository,
	KeeperConfig KeeperConfig,
) *Keeper {
	return &Keeper{
		updateCycle:          KeeperConfig.UpdateCycle,
		qualityFetchDebounce: KeeperConfig.QualityFetchDebounce,
		qualityOracleURL:     qualityOracleURL,
		oracleAPI:            NewOracleAPI(qualityOracleURL),
		countryProvider:      countryProvider,
	}
}

func (k *Keeper) StartAsync() {
	go k.start()
}

func (k *Keeper) start() {
	for {
		log.Info().Msg("proposal quality updated - started")

		countries, err := k.countryProvider.Countries()
		if err != nil {
			log.Err(err).Msg("failed updating proposal quality")
		}

		for _, country := range countries {
			k.sleepQualityFetchDebounce()
			qualities, err := k.oracleAPI.ProposalQualities(country)
			if err != nil {
				log.Err(err).Msgf("skipping proposal quality update for country: %s", country)
				continue
			}

			for _, q := range qualities.Entries {
				err := k.countryProvider.StoreQuality(
					q.ProposalID.ProviderID,
					q.ProposalID.ServiceType,
					country,
					q.Quality,
				)

				if err != nil {
					log.Err(err).Msgf("failed to store quality: %+w", q)
				}
			}
		}

		log.Info().Msg("proposal quality updated - completed")

		k.sleepUpdateCycle()
	}
}

func (k *Keeper) sleepUpdateCycle() {
	time.Sleep(k.updateCycle)
}

func (k *Keeper) sleepQualityFetchDebounce() {
	time.Sleep(k.qualityFetchDebounce)
}
