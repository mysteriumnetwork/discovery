package metrics

import (
	v2 "github.com/mysteriumnetwork/discovery/proposal/v2"
	"github.com/mysteriumnetwork/discovery/quality"
	"github.com/rs/zerolog/log"
)

type Enhancer struct {
	qualityService *quality.Service
}

func NewEnhancer(qualityService *quality.Service) *Enhancer {
	return &Enhancer{qualityService: qualityService}
}

func (s *Enhancer) EnhanceWithMetrics(resultMap map[string]*v2.Proposal, fromCountry string) {
	qualityResponse, err := s.qualityService.Quality(fromCountry)
	if err != nil {
		log.Warn().Err(err).Msgf("Could not fetch quality for consumer (country=%s)", fromCountry)
	} else {
		for _, q := range qualityResponse.Entries {
			key := q.ProposalID.ServiceType + q.ProposalID.ProviderID
			p, ok := resultMap[key]
			if !ok {
				continue
			}

			p.Quality.Quality = q.Quality
		}
	}

	latencyResponse, err := s.qualityService.Latency(fromCountry)
	if err != nil {
		log.Warn().Err(err).Msgf("Could not fetch quality for latency (country=%s)", fromCountry)
	} else {
		for _, latency := range latencyResponse.Entries {
			// latency does not have service type as it does not depend on it
			partialKey := latency.ProposalID.Key()

			for _, key := range []string{
				"noop" + partialKey,
				"wireguard" + partialKey,
				"openvpn" + partialKey,
			} {
				p, ok := resultMap[key]
				if ok {
					p.Quality.Latency = latency.Latency
				}
			}
		}
	}

	bandwidthResponse, err := s.qualityService.Bandwidth(fromCountry)
	if err != nil {
		log.Warn().Err(err).Msgf("Could not fetch quality for bandwidth (country=%s)", fromCountry)
	} else {
		for _, bandwidth := range bandwidthResponse.Entries {
			key := bandwidth.ProposalID.Key()
			p, ok := resultMap[key]
			if ok {
				p.Quality.Bandwidth = bandwidth.BandwidthMBPS
			}
		}
	}

}
