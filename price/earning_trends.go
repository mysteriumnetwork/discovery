package price

import (
	"math"
	"sort"
	"time"

	"github.com/mysteriumnetwork/discovery/price/pricingbyservice"
)

const stableTrendThreshold = 0.01

type EarningTrend string
type DemandIntensity string

const (
	EarningTrendIncreasing EarningTrend = "increasing"
	EarningTrendStable     EarningTrend = "stable"
	EarningTrendDecreasing EarningTrend = "decreasing"
)

const (
	DemandIntensityLow    DemandIntensity = "low"
	DemandIntensityMedium DemandIntensity = "medium"
	DemandIntensityHigh   DemandIntensity = "high"
)

type EarningTrendSummary struct {
	Increasing TrendSummaryItem `json:"increasing"`
	Stable     TrendSummaryItem `json:"stable"`
	Decreasing TrendSummaryItem `json:"decreasing"`
}

type TrendSummaryItem struct {
	Count           int     `json:"count"`
	SharePercentage float64 `json:"share_percentage"`
}

type CountryServiceTrend struct {
	CountryCode string                       `json:"country_code"`
	Country     string                       `json:"country"`
	Service     pricingbyservice.ServiceType `json:"service"`
	Trend       EarningTrend                 `json:"trend"`
	UpdatedAt   time.Time                    `json:"updated_at"`
	magnitude   float64
}

type CountryTrend struct {
	CountryCode string          `json:"country_code"`
	Country     string          `json:"country"`
	Trend       EarningTrend    `json:"trend"`
	Intensity   DemandIntensity `json:"intensity"`
	UpdatedAt   time.Time       `json:"updated_at"`
	change      float64
}

// CountryTrendView is the intentionally small country-only UI contract. It
// leaves service details, timestamps, numeric changes, and all prices on the
// server.
type CountryTrendView struct {
	CountryCode string          `json:"country_code"`
	Country     string          `json:"country"`
	Trend       EarningTrend    `json:"trend"`
	Intensity   DemandIntensity `json:"intensity"`
}

type EarningTrendsCountryResponse struct {
	Summary   EarningTrendSummary `json:"summary"`
	Countries []CountryTrendView  `json:"countries"`
}

type EarningTrendsResponse struct {
	Summary   EarningTrendSummary   `json:"summary"`
	Countries []CountryTrend        `json:"countries"`
	Trends    []CountryServiceTrend `json:"trends,omitempty"`
	Updates   []CountryServiceTrend `json:"latest_updates,omitempty"`
}

func countryEarningTrendsResponse(response EarningTrendsResponse) EarningTrendsCountryResponse {
	countries := make([]CountryTrendView, 0, len(response.Countries))
	for _, country := range response.Countries {
		countries = append(countries, CountryTrendView{
			CountryCode: country.CountryCode,
			Country:     country.Country,
			Trend:       country.Trend,
			Intensity:   country.Intensity,
		})
	}
	return EarningTrendsCountryResponse{
		Summary:   response.Summary,
		Countries: countries,
	}
}

type supportedTrendCountry struct {
	Code string
	Name string
}

var supportedTrendCountries = loadSupportedTrendCountries()

var trendServices = []pricingbyservice.ServiceType{
	pricingbyservice.ServiceTypeWireguard,
	pricingbyservice.ServiceTypeScraping,
	pricingbyservice.ServiceTypeQUICScraping,
	pricingbyservice.ServiceTypeDataTransfer,
	pricingbyservice.ServiceTypeDVPN,
	pricingbyservice.ServiceTypeMonitoring,
	pricingbyservice.ServiceTypeRuntime,
}

// earningTrends deliberately returns classifications only. Prices and the
// percentage by which an individual price changed never leave this function.
func earningTrends(prices pricingbyservice.LatestPrices) EarningTrendsResponse {
	result := EarningTrendsResponse{
		Countries: make([]CountryTrend, 0, len(supportedTrendCountries)),
		Trends:    make([]CountryServiceTrend, 0, len(supportedTrendCountries)*len(trendServices)),
		Updates:   make([]CountryServiceTrend, 0),
	}

	updatedAt := prices.PreviousValidUntil.UTC()
	for _, country := range supportedTrendCountries {
		code := country.Code
		history := prices.PerCountry[code]
		countryTrend, intensity, change := classifyCountryTrend(history, prices.Defaults)
		result.Countries = append(result.Countries, CountryTrend{
			CountryCode: code,
			Country:     country.Name,
			Trend:       countryTrend,
			Intensity:   intensity,
			UpdatedAt:   updatedAt,
			change:      change,
		})
		for _, service := range trendServices {
			change, ok := normalizedServiceChange(history, prices.Defaults, service)
			trend := classifyChange(change, ok)
			entry := CountryServiceTrend{
				CountryCode: code,
				Country:     country.Name,
				Service:     service,
				Trend:       trend,
				UpdatedAt:   updatedAt,
				magnitude:   math.Abs(change),
			}
			result.Trends = append(result.Trends, entry)
			if trend != EarningTrendStable {
				result.Updates = append(result.Updates, entry)
			}
		}
	}

	sort.Slice(result.Trends, func(i, j int) bool {
		if result.Trends[i].CountryCode == result.Trends[j].CountryCode {
			return result.Trends[i].Service < result.Trends[j].Service
		}
		return result.Trends[i].CountryCode < result.Trends[j].CountryCode
	})
	sort.Slice(result.Countries, func(i, j int) bool {
		// The country list powers "Biggest demand": strongest increases first,
		// followed by stable countries and then the strongest decreases.
		if result.Countries[i].change == result.Countries[j].change {
			return result.Countries[i].CountryCode < result.Countries[j].CountryCode
		}
		return result.Countries[i].change > result.Countries[j].change
	})
	sort.Slice(result.Updates, func(i, j int) bool {
		if result.Updates[i].magnitude == result.Updates[j].magnitude {
			if result.Updates[i].CountryCode == result.Updates[j].CountryCode {
				return result.Updates[i].Service < result.Updates[j].Service
			}
			return result.Updates[i].CountryCode < result.Updates[j].CountryCode
		}
		return result.Updates[i].magnitude > result.Updates[j].magnitude
	})
	result.Summary = summarizeCountries(result.Countries)
	return result
}

func classifyCountryTrend(history *pricingbyservice.PriceHistory, defaults *pricingbyservice.PriceHistory) (EarningTrend, DemandIntensity, float64) {
	if history == nil || history.Current == nil || history.Previous == nil || defaults == nil || defaults.Current == nil || defaults.Previous == nil {
		return EarningTrendStable, DemandIntensityLow, 0
	}
	// Each service contributes one vote to the country trend, regardless of
	// how many valid earning dimensions that service happens to contain.
	var total float64
	var count int
	for _, service := range trendServices {
		// Each service contributes once, regardless of how many of its price
		// dimensions are available, so sparse services are not underweighted.
		change, ok := normalizedServiceChange(history, defaults, service)
		if !ok {
			continue
		}
		total += change
		count++
	}
	if count == 0 {
		return EarningTrendStable, DemandIntensityLow, 0
	}
	average := total / float64(count)
	intensity := demandIntensity(math.Abs(average))
	if average > stableTrendThreshold {
		return EarningTrendIncreasing, intensity, average
	}
	if average < -stableTrendThreshold {
		return EarningTrendDecreasing, intensity, average
	}
	return EarningTrendStable, DemandIntensityLow, average
}

func demandIntensity(change float64) DemandIntensity {
	if change > 0.08 {
		return DemandIntensityHigh
	}
	if change > 0.03 {
		return DemandIntensityMedium
	}
	return DemandIntensityLow
}

func classifyTrend(current, previous, currentDefault, previousDefault float64) EarningTrend {
	change, ok := normalizedDimensionChange(current, previous, currentDefault, previousDefault)
	return classifyChange(change, ok)
}

func classifyChange(change float64, valid bool) EarningTrend {
	if !valid {
		return EarningTrendStable
	}
	if change > stableTrendThreshold {
		return EarningTrendIncreasing
	}
	if change < -stableTrendThreshold {
		return EarningTrendDecreasing
	}
	return EarningTrendStable
}

func normalizedServiceChange(history *pricingbyservice.PriceHistory, defaults *pricingbyservice.PriceHistory, service pricingbyservice.ServiceType) (float64, bool) {
	if history == nil || history.Current == nil || history.Previous == nil || defaults == nil || defaults.Current == nil || defaults.Previous == nil {
		return 0, false
	}

	// A service change is the mean of its valid Residential/Other and
	// per-GiB/per-hour dimensions. Each dimension is normalized to the same
	// dimension in the network default to remove network-wide price movement.
	// A service trend is the unweighted mean of every valid earning dimension:
	// Residential and Other, each for per-GiB and per-hour earnings. Dividing by
	// the matching network default isolates country demand from market-wide
	// price movement.
	priceTypes := [][4]*pricingbyservice.PriceByServiceType{
		{history.Current.Residential, history.Previous.Residential, defaults.Current.Residential, defaults.Previous.Residential},
		{history.Current.Other, history.Previous.Other, defaults.Current.Other, defaults.Previous.Other},
	}
	var total float64
	var count int
	for _, priceType := range priceTypes {
		current := servicePrice(priceType[0], service)
		previous := servicePrice(priceType[1], service)
		currentDefault := servicePrice(priceType[2], service)
		previousDefault := servicePrice(priceType[3], service)
		if current == nil || previous == nil || currentDefault == nil || previousDefault == nil {
			continue
		}
		for _, dimension := range [][4]float64{
			{current.PricePerGiBHumanReadable, previous.PricePerGiBHumanReadable, currentDefault.PricePerGiBHumanReadable, previousDefault.PricePerGiBHumanReadable},
			{current.PricePerHourHumanReadable, previous.PricePerHourHumanReadable, currentDefault.PricePerHourHumanReadable, previousDefault.PricePerHourHumanReadable},
		} {
			change, ok := normalizedDimensionChange(dimension[0], dimension[1], dimension[2], dimension[3])
			if !ok {
				continue
			}
			total += change
			count++
		}
	}
	if count == 0 {
		return 0, false
	}
	return total / float64(count), true
}

func normalizedDimensionChange(current, previous, currentDefault, previousDefault float64) (float64, bool) {
	// A current value of zero is a valid complete decrease. The previous and
	// default values must remain positive so the relative comparison is defined.
	if !isNonNegativeFinite(current) || !isPositiveFinite(previous) || !isPositiveFinite(currentDefault) || !isPositiveFinite(previousDefault) {
		return 0, false
	}
	change := (current/currentDefault)/(previous/previousDefault) - 1
	if math.IsNaN(change) || math.IsInf(change, 0) {
		return 0, false
	}
	return change, true
}

func isNonNegativeFinite(value float64) bool {
	return value >= 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func isPositiveFinite(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func servicePrice(prices *pricingbyservice.PriceByServiceType, service pricingbyservice.ServiceType) *pricingbyservice.Price {
	if prices == nil {
		return nil
	}
	switch service {
	case pricingbyservice.ServiceTypeWireguard:
		return &prices.Wireguard
	case pricingbyservice.ServiceTypeScraping:
		return &prices.Scraping
	case pricingbyservice.ServiceTypeQUICScraping:
		return &prices.QUICScraping
	case pricingbyservice.ServiceTypeDataTransfer:
		return &prices.DataTransfer
	case pricingbyservice.ServiceTypeDVPN:
		return &prices.DVPN
	case pricingbyservice.ServiceTypeMonitoring:
		return &prices.Monitoring
	case pricingbyservice.ServiceTypeRuntime:
		return &prices.Runtime
	default:
		return nil
	}
}

func loadSupportedTrendCountries() []supportedTrendCountry {
	// Pricing support, not map resolution, defines the analytics universe.
	// Antarctica is excluded because it is not a node-running market. Small
	// countries and territories that are not visible in the low-resolution map
	// remain available in the summary and accessible country list.
	countries := make([]supportedTrendCountry, 0, len(pricingbyservice.CountryCodeToName)-1)
	for code, name := range pricingbyservice.CountryCodeToName {
		if code == "AQ" {
			continue
		}
		countries = append(countries, supportedTrendCountry{Code: code.String(), Name: name})
	}
	sort.Slice(countries, func(i, j int) bool { return countries[i].Code < countries[j].Code })
	return countries
}

func summarizeTrends(trends []CountryServiceTrend) EarningTrendSummary {
	var summary EarningTrendSummary
	for _, trend := range trends {
		switch trend.Trend {
		case EarningTrendIncreasing:
			summary.Increasing.Count++
		case EarningTrendDecreasing:
			summary.Decreasing.Count++
		default:
			summary.Stable.Count++
		}
	}
	if len(trends) == 0 {
		return summary
	}
	total := float64(len(trends))
	summary.Increasing.SharePercentage = roundPercentage(float64(summary.Increasing.Count) / total * 100)
	summary.Stable.SharePercentage = roundPercentage(float64(summary.Stable.Count) / total * 100)
	summary.Decreasing.SharePercentage = roundPercentage(float64(summary.Decreasing.Count) / total * 100)
	return summary
}

func summarizeCountries(countries []CountryTrend) EarningTrendSummary {
	trends := make([]CountryServiceTrend, 0, len(countries))
	for _, country := range countries {
		trends = append(trends, CountryServiceTrend{Trend: country.Trend})
	}
	return summarizeTrends(trends)
}

func roundPercentage(value float64) float64 {
	return math.Round(value*10) / 10
}
