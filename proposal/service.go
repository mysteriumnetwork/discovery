// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package proposal

import (
	"time"

	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/discovery/proposal/metrics"
	v3 "github.com/mysteriumnetwork/discovery/proposal/v3"
	"github.com/mysteriumnetwork/discovery/quality"
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
	from                    string
	providerIDS             []string
	serviceType             string
	locationCountry         string
	ipType                  string
	accessPolicy            string
	accessPolicySource      string
	compatibilityMin        int
	compatibilityMax        int
	qualityMin              float64
	tags                    string
	includeMonitoringFailed bool
	natCompatibility        string
}

func (s *Service) List(opts ListOpts) ([]v3.Proposal, error) {
	proposals, err := s.Repository.List(repoListOpts{
		providerIDS:        opts.providerIDS,
		serviceType:        opts.serviceType,
		country:            opts.locationCountry,
		ipType:             opts.ipType,
		accessPolicy:       opts.accessPolicy,
		accessPolicySource: opts.accessPolicySource,
		compatibilityMin:   opts.compatibilityMin,
		compatibilityMax:   opts.compatibilityMax,
		tags:               opts.tags,
	})
	if err != nil {
		return nil, err
	}
	resultMap := make(map[string]*v3.Proposal, len(proposals))
	for i, p := range proposals {
		resultMap[p.ServiceType+p.ProviderID] = &proposals[i]
	}

	or := &metrics.OracleResponses{}
	or.Load(s.qualityService, opts.from)

	metrics.EnhanceWithMetrics(resultMap, or, metrics.Filters{
		IncludeMonitoringFailed: opts.includeMonitoringFailed,
		NatCompatibility:        opts.natCompatibility,
	})

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

func values(proposalsMap map[string]*v3.Proposal) []v3.Proposal {
	res := make([]v3.Proposal, 0)
	for k := range proposalsMap {
		res = append(res, *proposalsMap[k])
	}
	return res
}

func (s *Service) Shutdown() {
	s.shutdown <- struct{}{}
}
