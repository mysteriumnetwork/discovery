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

func (s *Service) List(opts ListOpts) []v3.Proposal {
	proposals := s.Repository.List(repoListOpts{
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

	or := &metrics.OracleResponses{}
	or.Load(s.qualityService, opts.from)

	return metrics.EnhanceWithMetrics(proposals, or.QualityResponse, metrics.Filters{
		IncludeMonitoringFailed: opts.includeMonitoringFailed,
		NATCompatibility:        opts.natCompatibility,
		QualityMin:              opts.qualityMin,
	})
}

func (s *Service) Metadata(opts repoMetadataOpts) []v3.Metadata {
	or := &metrics.OracleResponses{}
	or.Load(s.qualityService, "US")

	return s.Repository.Metadata(opts, or.QualityResponse)
}

func (s *Service) ListCountriesNumbers(opts ListOpts) map[string]int {
	return s.Repository.ListCountriesNumbers(repoListOpts{
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
}

func (s *Service) StartExpirationJob() {
	for {
		select {
		case <-time.After(s.expirationJobDelay):
			count := s.Repository.Expire()
			log.Debug().Msgf("Expired proposals: %v", count)
		case <-s.shutdown:
			return
		}
	}
}

func (s *Service) Shutdown() {
	s.shutdown <- struct{}{}
}
