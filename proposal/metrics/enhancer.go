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
		log.Error().Err(err).Msgf("Could not fetch quality for consumer (country=%s)", fromCountry)
	}
	or.QualityResponse = qRes
}

type Filters struct {
	IncludeMonitoringFailed bool
	NATCompatibility        string
	QualityMin              float64
	BandwidthMin            float64
	PresetID                int
}

var emptyQuality = oracleapi.DetailedQuality{
	MonitoringFailed: true,
	RestrictedNode:   true,
}

func EnhanceWithMetrics(proposals []v3.Proposal, or map[string]*oracleapi.DetailedQuality, f Filters) (res []v3.Proposal) {
	for _, p := range proposals {
		key := p.Key()

		if len(or) == 0 {
			res = append(res, p)
			continue
		}

		q, ok := or[key]
		if !ok {
			q = &emptyQuality
		}

		if f.NATCompatibility == "symmetric" && q.RestrictedNode {
			continue
		}

		if !f.IncludeMonitoringFailed && q.MonitoringFailed {
			continue
		}

		if q.Quality < f.QualityMin {
			continue
		}

		if f.BandwidthMin > 0 && q.Bandwidth < f.BandwidthMin {
			continue
		}

		p.Quality.Quality = q.Quality
		p.Quality.Latency = q.Latency
		p.Quality.Bandwidth = q.Bandwidth
		p.Quality.Uptime = q.Uptime
		p.Quality.MonitoringFailed = q.MonitoringFailed

		if !matchPreset(f.PresetID, p) {
			continue
		}

		res = append(res, p)
	}

	return res
}

func matchPreset(presetID int, p v3.Proposal) bool {
	switch presetID {
	case 1:
		if p.Location.IPType != "residential" || p.Quality.Quality < 1 || p.Quality.Bandwidth < 5 {
			return false
		}
	case 2:
		if p.Quality.Quality < 1 {
			return false
		}
	case 3:
		if p.Location.IPType != "hosting" {
			return false
		}
	}

	return true
}
