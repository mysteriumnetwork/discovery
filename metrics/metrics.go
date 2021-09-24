package metrics

import "github.com/prometheus/client_golang/prometheus"

var CurrentPriceByCountry = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "discovery_current_price",
		Help: "Current pricing by country",
	},
	[]string{"country_code", "node_type", "price_type"},
)

func InitialiseMonitoring() {
	prometheus.MustRegister(
		CurrentPriceByCountry,
	)
}
