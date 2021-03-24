package quality

import (
	"time"

	"github.com/mysteriumnetwork/discovery/location"
	"github.com/mysteriumnetwork/discovery/quality/oracleapi"
	"github.com/rs/zerolog/log"
)

type Updater struct {
	UpdaterOpts
	repository *Repository
	api        *oracleapi.API
}

type UpdaterOpts struct {
	UpdateCycleDelay  time.Duration
	QualityFetchDelay time.Duration
}

func NewUpdater(
	qualityOracleURL string,
	qualityRepository *Repository,
	opts UpdaterOpts,
) *Updater {
	return &Updater{
		UpdaterOpts: opts,
		api:         oracleapi.New(qualityOracleURL),
		repository:  qualityRepository,
	}
}

func (u *Updater) Start() {
	for {
		log.Info().Msg("Quality updater: started")

		for _, country := range location.Countries {
			time.Sleep(u.QualityFetchDelay)

			qualities, err := u.api.Quality(country)
			if err != nil {
				log.Err(err).Msgf("Failed to fetch quality (country=%s)", country)
				continue
			}

			for _, q := range qualities.Entries {
				err := u.repository.StoreQuality(
					q.ProposalID.ProviderID,
					q.ProposalID.ServiceType,
					country,
					q.Quality,
				)

				if err != nil {
					log.Err(err).Msgf("Failed to store quality: %+v", q)
				}
			}
		}

		log.Info().Msg("Quality updater: cycle complete")

		time.Sleep(u.UpdateCycleDelay)
	}
}
