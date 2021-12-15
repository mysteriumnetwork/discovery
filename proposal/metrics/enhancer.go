package metrics

import (
	"github.com/rs/zerolog/log"

	v3 "github.com/mysteriumnetwork/discovery/proposal/v3"
	"github.com/mysteriumnetwork/discovery/quality"
	"github.com/mysteriumnetwork/discovery/quality/oracleapi"
)

type OracleResponses struct {
	QualityResponse map[string]*oracleapi.DetailedQuality
}

func (or *OracleResponses) Load(qualityService *quality.Service, fromCountry string) {
	qRes, err := qualityService.Quality(fromCountry)
	if err != nil {
		log.Warn().Err(err).Msgf("Could not fetch quality for consumer (country=%s)", fromCountry)
	}
	or.QualityResponse = qRes
}

type Filters struct {
	IncludeMonitoringFailed bool
	NATCompatibility        string
	QualityMin              float64
}

func EnhanceWithMetrics(proposals []v3.Proposal, or map[string]*oracleapi.DetailedQuality, f Filters) (res []v3.Proposal) {
	for _, p := range proposals {
		key := p.Key()

		q, ok := or[key]
		if !ok {
			continue
		}

		if f.NATCompatibility == "symmetric" && q.RestrictedNode {
			continue
		}

		if !f.IncludeMonitoringFailed && q.MonitoringFailed {
			continue
		}

		if p.Quality.Quality < f.QualityMin {
			continue
		}

		p.Quality.Quality = q.Quality
		p.Quality.Latency = q.Latency
		p.Quality.Bandwidth = q.Bandwidth
		p.Quality.Bandwidth = q.Bandwidth
		p.Quality.MonitoringFailed = q.MonitoringFailed
		res = append(res, p)
	}

	return res
}
