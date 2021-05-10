// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package proposal

import (
	"time"

	"github.com/mysteriumnetwork/discovery/proposal/metrics"

	v2 "github.com/mysteriumnetwork/discovery/proposal/v2"
	"github.com/mysteriumnetwork/discovery/quality"
	"github.com/rs/zerolog/log"
)

type Service struct {
	*Repository
	qualityService *quality.Service
	shutdown       chan struct{}
	enhancer       *metrics.Enhancer
}

func NewService(repository *Repository, qualityService *quality.Service) *Service {
	return &Service{
		Repository:     repository,
		qualityService: qualityService,
		enhancer:       metrics.NewEnhancer(qualityService),
	}
}

type ListOpts struct {
	from                      string
	providerID                string
	serviceType               string
	locationCountry           string
	ipType                    string
	accessPolicy              string
	accessPolicySource        string
	compatibilityMin          int
	compatibilityMax          int
	qualityMin                float64
	priceGiBMax, priceHourMax int64
}

func (s *Service) List(opts ListOpts) ([]v2.Proposal, error) {
	proposals, err := s.Repository.List(repoListOpts{
		providerID:         opts.providerID,
		serviceType:        opts.serviceType,
		country:            opts.locationCountry,
		ipType:             opts.ipType,
		accessPolicy:       opts.accessPolicy,
		accessPolicySource: opts.accessPolicySource,
		compatibilityMin:   opts.compatibilityMin,
		compatibilityMax:   opts.compatibilityMax,
		priceGiBMax:        opts.priceGiBMax,
		priceHourMax:       opts.priceHourMax,
	})
	if err != nil {
		return nil, err
	}
	resultMap := make(map[string]*v2.Proposal, len(proposals))
	for i, p := range proposals {
		resultMap[p.ServiceType+p.ProviderID] = &proposals[i]
	}

	// filter by quality
	qualityResponse, err := s.qualityService.Quality(opts.from)
	if err != nil {
		log.Warn().Err(err).Msgf("Could not fetch quality for consumer (from country=%s)", opts.from)
		return values(resultMap), nil
	}

	for _, q := range qualityResponse.Entries {
		key := q.ProposalID.ServiceType + q.ProposalID.ProviderID
		p, ok := resultMap[key]
		if !ok {
			continue
		}

		if opts.qualityMin > q.Quality {
			delete(resultMap, key)
			continue
		}

		p.Quality.Quality = q.Quality
	}

	// exclude monitoringFailed nodes
	sessionsResponse, err := s.qualityService.Sessions()
	if err != nil {
		log.Warn().Err(err).Msgf("Could not fetch session stats for consumer", opts.from)
		return values(resultMap), nil
	}

	for k, proposal := range resultMap {
		if sessionsResponse.MonitoringFailed(proposal.ProviderID, proposal.ServiceType) {
			delete(resultMap, k)
		}
	}

	s.enhancer.EnhanceWithMetrics(resultMap, opts.from)
	for key, proposal := range resultMap {
		if proposal.Quality.Quality < opts.qualityMin {
			delete(resultMap, key)
		}
	}

	return values(resultMap), nil
}

func (s *Service) StartExpirationJob() {
	for {
		select {
		case <-time.After(s.expirationJobDelay):
			log.Debug().Msg("Running expiration job")
			count, err := s.Repository.Expire()
			if err != nil {
				log.Err(err).Msg("Failed to expire proposals")
			} else {
				log.Debug().Msgf("Expired proposals: %v", count)
			}
		case <-s.shutdown:
			return
		}
	}
}

func values(proposalsMap map[string]*v2.Proposal) []v2.Proposal {
	var res = make([]v2.Proposal, 0)
	for k := range proposalsMap {
		res = append(res, *proposalsMap[k])
	}
	return res
}

func (s *Service) Shutdown() {
	s.shutdown <- struct{}{}
}
