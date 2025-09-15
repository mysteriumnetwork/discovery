package aggregate

import (
	"github.com/mysteriumnetwork/discovery/quality/oracleapi"
)

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

func EnhanceWithMetrics(proposals []Proposal, or map[string]*oracleapi.DetailedQuality, f Filters) (res []Proposal) {
	for _, p := range proposals {
		if len(or) == 0 {
			res = append(res, p)
			continue
		}

		q := &emptyQuality
		for _, s := range []string{"data_transfer", "scraping", "wireguard", "dvpn", "monitoring"} {
			if pq, ok := or[p.ProviderID+"."+s]; ok {
				q = pq
				break
			}
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
		p.Quality.PacketLoss = q.PacketLoss
		p.Quality.MonitoringFailed = q.MonitoringFailed

		if !matchPreset(f.PresetID, p) {
			continue
		}

		res = append(res, p)
	}

	return res
}

func matchPreset(presetID int, p Proposal) bool {
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
