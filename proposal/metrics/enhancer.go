package metrics

import (
	"sync"

	v3 "github.com/mysteriumnetwork/discovery/proposal/v3"
	"github.com/mysteriumnetwork/discovery/quality"
	"github.com/mysteriumnetwork/discovery/quality/oracleapi"
	"github.com/rs/zerolog/log"
)

type OracleResponses struct {
	QualityResponse   *oracleapi.ProposalQualityResponse
	LatencyResponse   *oracleapi.LatencyResponse
	BandwitdhResponse *oracleapi.BandwidthResponse
	SessionResponse   *oracleapi.SessionsResponse
}

func (or *OracleResponses) Load(qualityService *quality.Service, fromCountry string) {
	wg := sync.WaitGroup{}
	wg.Add(4)

	go func() {
		defer wg.Done()
		qRes, err := qualityService.Quality(fromCountry)
		if err != nil {
			log.Warn().Err(err).Msgf("Could not fetch quality for consumer (country=%s)", fromCountry)
		}
		or.QualityResponse = qRes
	}()

	go func() {
		defer wg.Done()
		latencyRes, err := qualityService.Latency(fromCountry)
		if err != nil {
			log.Warn().Err(err).Msgf("Could not fetch quality for latency (country=%s)", fromCountry)
		}
		or.LatencyResponse = latencyRes
	}()

	go func() {
		defer wg.Done()
		bandwidthRes, err := qualityService.Bandwidth(fromCountry)
		if err != nil {
			log.Warn().Err(err).Msgf("Could not fetch quality for bandwidth (country=%s)", fromCountry)
		}
		or.BandwitdhResponse = bandwidthRes
	}()

	go func() {
		defer wg.Done()
		sessionRes, err := qualityService.Sessions(fromCountry)
		if err != nil {
			log.Warn().Err(err).Msgf("Could not fetch quality for sessions (country=%s)", fromCountry)
		}
		or.SessionResponse = sessionRes
	}()

	wg.Wait()
}

type Filters struct {
	QualityMin              float64
	IncludeMonitoringFailed bool
	NatCompatibility        string
}

func EnhanceWithMetrics(resultMap map[string]*v3.Proposal, or *OracleResponses, f Filters) {
	if or.QualityResponse != nil {
		for _, q := range or.QualityResponse.Entries {
			key := q.ProposalID.ServiceType + q.ProposalID.ProviderID
			p, ok := resultMap[key]
			if !ok {
				continue
			}

			if q.Quality < f.QualityMin {
				delete(resultMap, key)
				continue
			}

			if f.NatCompatibility == "symmetric" && q.RestrictedNode {
				delete(resultMap, key)
				continue
			}

			p.Quality.Quality = q.Quality
		}
	}

	if or.LatencyResponse != nil {
		for _, latency := range or.LatencyResponse.Entries {
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

	if or.BandwitdhResponse != nil {
		for _, bandwidth := range or.BandwitdhResponse.Entries {
			key := bandwidth.ProposalID.Key()
			p, ok := resultMap[key]
			if ok {
				p.Quality.Bandwidth = bandwidth.BandwidthMBPS
			}
		}
	}

	if or.SessionResponse != nil {
		for k, proposal := range resultMap {

			if !f.IncludeMonitoringFailed && or.SessionResponse.MonitoringFailed(proposal.ProviderID, proposal.ServiceType) {
				delete(resultMap, k)
				continue
			}

			proposal.Quality.MonitoringFailed = or.SessionResponse.MonitoringFailedOrNil(proposal.ProviderID, proposal.ServiceType)
		}
	}
}
