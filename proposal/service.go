// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package proposal

import (
	"time"

	v2 "github.com/mysteriumnetwork/discovery/proposal/v2"
	"github.com/mysteriumnetwork/discovery/quality"
	"github.com/rs/zerolog/log"
)

type Service struct {
	*Repository
	qualityService *quality.Service
	shutdown       chan struct{}
}

func NewService(repository *Repository, qualityService *quality.Service) *Service {
	return &Service{
		Repository:     repository,
		qualityService: qualityService,
	}
}

type ListOpts struct {
	from                 string
	serviceType, country string
	residential          bool
	accessPolicy         string
}

func (s *Service) List(opts ListOpts) ([]v2.Proposal, error) {
	proposals, err := s.Repository.List(repoListOpts{
		serviceType:  opts.serviceType,
		country:      opts.country,
		residential:  opts.residential,
		accessPolicy: opts.accessPolicy,
	})
	if err != nil {
		return nil, err
	}
	resultMap := make(map[string]*v2.Proposal, len(proposals))
	for i, p := range proposals {
		resultMap[p.ServiceType+p.ProviderID] = &proposals[i]
	}

	qualityResponse, err := s.qualityService.Quality(opts.from)
	if err != nil {
		log.Warn().Err(err).Msgf("Could not fetch quality for consumer (country=%s)", opts.from)
		return values(resultMap), nil
	}

	for _, q := range qualityResponse.Entries {
		p, ok := resultMap[q.ProposalID.ServiceType+q.ProposalID.ProviderID]
		if !ok {
			continue
		}
		p.Quality = q.Quality
	}

	return values(resultMap), nil
}

func (s *Service) StartExpirationJob() {
	for {
		select {
		case <-time.After(s.expirationJobDelay):
			log.Debug().Msgf("Running expiration job")
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
