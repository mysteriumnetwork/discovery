package price

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/html"

	"github.com/mysteriumnetwork/discovery/price/pricingbyservice"
)

func TestEarningTrendsClassifiesCountryAndServiceWithoutExposingPrices(t *testing.T) {
	updatedAt := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	prices := pricingbyservice.LatestPrices{
		Defaults: history(servicePrices(100, 100, 100), servicePrices(200, 200, 200)),
		PerCountry: map[string]*pricingbyservice.PriceHistory{
			"US": history(servicePrices(110, 100.5, 80), servicePrices(200, 200, 200)),
		},
		PreviousValidUntil: updatedAt,
	}

	result := earningTrends(prices)
	byService := make(map[pricingbyservice.ServiceType]EarningTrend)
	for _, trend := range result.Trends {
		if trend.CountryCode == "US" {
			byService[trend.Service] = trend.Trend
		}
	}

	if got := byService[pricingbyservice.ServiceTypeWireguard]; got != EarningTrendIncreasing {
		t.Fatalf("wireguard trend = %q, want increasing", got)
	}
	if got := byService[pricingbyservice.ServiceTypeScraping]; got != EarningTrendStable {
		t.Fatalf("scraping trend = %q, want stable", got)
	}
	if got := byService[pricingbyservice.ServiceTypeDVPN]; got != EarningTrendDecreasing {
		t.Fatalf("dvpn trend = %q, want decreasing", got)
	}
	if len(result.Updates) != 2 {
		t.Fatalf("latest updates = %d, want 2", len(result.Updates))
	}
	if result.Updates[0].Service != pricingbyservice.ServiceTypeDVPN {
		t.Fatalf("largest update = %q, want dvpn", result.Updates[0].Service)
	}

	body, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"price_per_gib", "price_per_hour", "change_percentage", "magnitude"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("response exposes forbidden field %q: %s", forbidden, body)
		}
	}
}

func TestEarningTrendsUsesCurrentNetworkBaseline(t *testing.T) {
	prices := pricingbyservice.LatestPrices{
		Defaults: history(servicePrices(100, 100, 100), servicePrices(200, 200, 200)),
		PerCountry: map[string]*pricingbyservice.PriceHistory{
			"US": history(servicePrices(100, 100, 100), servicePrices(300, 300, 300)),
		},
	}

	for _, trend := range earningTrends(prices).Trends {
		if trend.Trend != EarningTrendStable {
			t.Fatalf("%s trend = %q, want stable", trend.Service, trend.Trend)
		}
	}
}

func TestEarningTrendsKeepsActiveBoostVisibleWhenPreviousMatchesCurrent(t *testing.T) {
	prices := pricingbyservice.LatestPrices{
		Defaults: history(uniformPrices(1), uniformPrices(1)),
		PerCountry: map[string]*pricingbyservice.PriceHistory{
			"BG": history(uniformPrices(1.2), uniformPrices(1.2)),
		},
	}

	result := earningTrends(prices)
	for _, country := range result.Countries {
		if country.CountryCode == "BG" {
			if country.Trend != EarningTrendIncreasing || country.Intensity != DemandIntensityHigh {
				t.Fatalf("BG classification = (%q, %q), want increasing/high", country.Trend, country.Intensity)
			}
			return
		}
	}
	t.Fatal("BG is missing from country trends")
}

func TestCountryTrendUsesAverageAcrossServices(t *testing.T) {
	defaults := history(servicePrices(100, 100, 100), servicePrices(100, 100, 100))
	country := history(servicePrices(121, 91, 100), servicePrices(100, 100, 100))
	if got, intensity, _ := classifyCountryTrend(country, defaults); got != EarningTrendIncreasing || intensity != DemandIntensityMedium {
		t.Fatalf("country trend = %q, want increasing", got)
	}

	noData := &pricingbyservice.PriceHistory{
		Current:  &pricingbyservice.PriceByType{},
		Previous: &pricingbyservice.PriceByType{},
	}
	if got, intensity, _ := classifyCountryTrend(noData, defaults); got != EarningTrendStable || intensity != DemandIntensityLow {
		t.Fatalf("no-data country trend = %q, want stable", got)
	}
}

func TestNormalizedServiceChangeUsesBothPriceTypesAndEarningDimensions(t *testing.T) {
	defaults := history(
		wireguardPrices(100, 100, 100, 100),
		wireguardPrices(100, 100, 100, 100),
	)
	country := history(
		wireguardPrices(130, 110, 90, 90),
		wireguardPrices(100, 100, 100, 100),
	)

	change, ok := normalizedServiceChange(country, defaults, pricingbyservice.ServiceTypeWireguard)
	if !ok {
		t.Fatal("wireguard change is not valid")
	}
	if math.Abs(change-0.05) > 1e-9 {
		t.Fatalf("wireguard change = %v, want 0.05", change)
	}
}

func TestCountryTrendWeightsServicesEquallyWhenDimensionsAreSparse(t *testing.T) {
	fullWireguard := func(wireguardMultiplier, scrapingMultiplier float64) *pricingbyservice.PriceByType {
		return &pricingbyservice.PriceByType{
			Residential: &pricingbyservice.PriceByServiceType{
				Wireguard: pricingbyservice.Price{
					PricePerGiBHumanReadable:  100 * wireguardMultiplier,
					PricePerHourHumanReadable: 100 * wireguardMultiplier,
				},
				Scraping: pricingbyservice.Price{PricePerGiBHumanReadable: 100 * scrapingMultiplier},
			},
			Other: &pricingbyservice.PriceByServiceType{
				Wireguard: pricingbyservice.Price{
					PricePerGiBHumanReadable:  100 * wireguardMultiplier,
					PricePerHourHumanReadable: 100 * wireguardMultiplier,
				},
			},
		}
	}
	defaults := history(fullWireguard(1, 1), fullWireguard(1, 1))
	country := history(fullWireguard(1.2, 0.9), fullWireguard(1, 1))

	trend, intensity, change := classifyCountryTrend(country, defaults)
	if trend != EarningTrendIncreasing || intensity != DemandIntensityMedium {
		t.Fatalf("country classification = (%q, %q), want (increasing, medium)", trend, intensity)
	}
	if math.Abs(change-0.05) > 1e-9 {
		t.Fatalf("country change = %v, want equally weighted service average 0.05", change)
	}
}

func TestInvalidOrMissingPriceDimensionsAreStable(t *testing.T) {
	defaults := history(
		wireguardPrices(100, 100, 100, 100),
		wireguardPrices(100, 100, 100, 100),
	)
	invalid := history(
		wireguardPrices(math.NaN(), -1, math.Inf(1), -1),
		wireguardPrices(100, 100, 100, 100),
	)

	if change, ok := normalizedServiceChange(invalid, defaults, pricingbyservice.ServiceTypeWireguard); ok || change != 0 {
		t.Fatalf("invalid service change = (%v, %v), want (0, false)", change, ok)
	}
	if trend, intensity, _ := classifyCountryTrend(invalid, defaults); trend != EarningTrendStable || intensity != DemandIntensityLow {
		t.Fatalf("invalid country classification = (%q, %q), want stable/low", trend, intensity)
	}
}

func TestZeroCurrentEarningsAreACompleteDecrease(t *testing.T) {
	defaults := history(
		wireguardPrices(100, 100, 100, 100),
		wireguardPrices(100, 100, 100, 100),
	)
	country := history(
		wireguardPrices(0, 0, 0, 0),
		wireguardPrices(100, 100, 100, 100),
	)

	change, ok := normalizedServiceChange(country, defaults, pricingbyservice.ServiceTypeWireguard)
	if !ok || change != -1 {
		t.Fatalf("zero-current change = (%v, %v), want (-1, true)", change, ok)
	}
	if trend, intensity, got := classifyCountryTrend(country, defaults); trend != EarningTrendDecreasing || intensity != DemandIntensityHigh || got != -1 {
		t.Fatalf("zero-current classification = (%q, %q, %v), want decreasing/high/-1", trend, intensity, got)
	}
}

func TestDemandIntensityDoesNotExposeNumericChange(t *testing.T) {
	if demandIntensity(0.02) != DemandIntensityLow || demandIntensity(0.05) != DemandIntensityMedium || demandIntensity(0.20) != DemandIntensityHigh {
		t.Fatal("unexpected demand intensity classification")
	}
}

func TestEarningTrendsEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api := &APIByService{pricer: staticLatestPricer{prices: pricingbyservice.LatestPrices{}}}
	router := gin.New()
	router.GET("/api/v4/earning-trends", api.EarningTrends)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v4/earning-trends", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if strings.Contains(response.Body.String(), "price") {
		t.Fatalf("response contains price data: %s", response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("cache control = %q, want no-store", got)
	}
	if got := response.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
}

func TestEarningTrendsCountryScopeOmitsServiceDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api := &APIByService{pricer: staticLatestPricer{prices: pricingbyservice.LatestPrices{}}}
	router := gin.New()
	router.GET("/api/v4/earning-trends", api.EarningTrends)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v4/earning-trends?scope=countries", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	for _, required := range []string{"summary", "countries"} {
		if _, ok := payload[required]; !ok {
			t.Fatalf("country-scoped response is missing %q: %s", required, response.Body.String())
		}
	}
	for _, omitted := range []string{"trends", "latest_updates", "demo_note"} {
		if _, ok := payload[omitted]; ok {
			t.Fatalf("country-scoped response includes %q: %s", omitted, response.Body.String())
		}
	}
	if strings.Contains(response.Body.String(), "updated_at") {
		t.Fatalf("country-scoped response includes unused dates: %s", response.Body.String())
	}

	var summary EarningTrendSummary
	if err := json.Unmarshal(payload["summary"], &summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	var countries []CountryTrendView
	if err := json.Unmarshal(payload["countries"], &countries); err != nil {
		t.Fatalf("decode countries: %v", err)
	}
	if len(countries) != len(supportedTrendCountries) {
		t.Fatalf("countries = %d, want %d", len(countries), len(supportedTrendCountries))
	}
	if got := summary.Increasing.Count + summary.Stable.Count + summary.Decreasing.Count; got != len(countries) {
		t.Fatalf("summary count = %d, countries = %d", got, len(countries))
	}
}

func TestEarningTrendsWithoutPricesReturnsEverySupportedCountryAsStable(t *testing.T) {
	result := earningTrends(pricingbyservice.LatestPrices{})
	if len(supportedTrendCountries) == 0 {
		t.Fatal("supported country universe is empty")
	}
	if len(result.Countries) != len(supportedTrendCountries) {
		t.Fatalf("countries = %d, want %d", len(result.Countries), len(supportedTrendCountries))
	}
	if result.Summary.Stable.Count != len(supportedTrendCountries) || result.Summary.Increasing.Count != 0 || result.Summary.Decreasing.Count != 0 {
		t.Fatalf("empty-price summary = %+v", result.Summary)
	}
	if result.Summary.Stable.SharePercentage != 100 {
		t.Fatalf("stable share = %v, want 100", result.Summary.Stable.SharePercentage)
	}
	for _, country := range result.Countries {
		if country.Trend != EarningTrendStable || country.Intensity != DemandIntensityLow {
			t.Fatalf("%s fallback = (%q, %q), want stable/low", country.CountryCode, country.Trend, country.Intensity)
		}
	}
	if len(result.Trends) != len(supportedTrendCountries)*len(trendServices) {
		t.Fatalf("country-service trends = %d, want %d", len(result.Trends), len(supportedTrendCountries)*len(trendServices))
	}
	for _, trend := range result.Trends {
		if trend.Trend != EarningTrendStable {
			t.Fatalf("%s/%s no-data trend = %q, want stable", trend.CountryCode, trend.Service, trend.Trend)
		}
	}
}

func TestSupportedTrendCountriesCoverPricingUniverseExceptAntarctica(t *testing.T) {
	if len(supportedTrendCountries) != len(pricingbyservice.CountryCodeToName)-1 {
		t.Fatalf("supported countries = %d, want %d", len(supportedTrendCountries), len(pricingbyservice.CountryCodeToName)-1)
	}
	seen := make(map[string]struct{}, len(supportedTrendCountries))
	for _, country := range supportedTrendCountries {
		if len(country.Code) != 2 || country.Code == "AQ" || country.Code == "-99" || country.Name == "" {
			t.Fatalf("unsupported map country included: %+v", country)
		}
		if _, ok := seen[country.Code]; ok {
			t.Fatalf("duplicate map country %q", country.Code)
		}
		seen[country.Code] = struct{}{}
	}
	if _, ok := seen["UA"]; !ok {
		t.Fatal("Ukraine is missing from supported countries")
	}
	for _, code := range []string{"SG", "HK", "MT"} {
		if _, ok := seen[code]; !ok {
			t.Fatalf("pricing-supported country %s is missing", code)
		}
	}
}

func TestCountriesAreRankedBySignedDemandChange(t *testing.T) {
	defaults := history(uniformPrices(1), uniformPrices(1))
	prices := pricingbyservice.LatestPrices{
		Defaults: defaults,
		PerCountry: map[string]*pricingbyservice.PriceHistory{
			"US": history(uniformPrices(1.2), uniformPrices(1)),
			"IT": history(uniformPrices(1.005), uniformPrices(1)),
			"FR": history(uniformPrices(1), uniformPrices(1)),
			"GB": history(uniformPrices(0.995), uniformPrices(1)),
			"PL": history(uniformPrices(0.8), uniformPrices(1)),
		},
	}

	result := earningTrends(prices)
	positions := make(map[string]int, len(result.Countries))
	for index, country := range result.Countries {
		positions[country.CountryCode] = index
	}
	orderedCodes := []string{"US", "IT", "FR", "GB", "PL"}
	for index := 1; index < len(orderedCodes); index++ {
		previous, current := orderedCodes[index-1], orderedCodes[index]
		if positions[previous] >= positions[current] {
			t.Fatalf("signed ranking puts %s at %d and %s at %d", previous, positions[previous], current, positions[current])
		}
	}
	if result.Countries[0].CountryCode != "US" || result.Countries[len(result.Countries)-1].CountryCode != "PL" {
		t.Fatalf("ranking endpoints = %s ... %s, want US ... PL", result.Countries[0].CountryCode, result.Countries[len(result.Countries)-1].CountryCode)
	}
}

func TestEarningTrendsUI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api := &APIByService{}
	router := gin.New()
	router.GET("/api/v4/earning-trends/view", api.EarningTrendsUI)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v4/earning-trends/view", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Fatalf("content type = %q", contentType)
	}

	document, err := html.Parse(strings.NewReader(response.Body.String()))
	if err != nil {
		t.Fatalf("parse UI HTML: %v", err)
	}

	stylesheet := requireHTMLElement(t, document, "link", "")
	if relationship, _ := htmlAttribute(stylesheet, "rel"); relationship != "stylesheet" {
		t.Fatalf("first link rel = %q, want stylesheet", relationship)
	}
	if href, _ := htmlAttribute(stylesheet, "href"); href != "/api/v4/earning-trends/assets/view.css" {
		t.Fatalf("stylesheet href = %q", href)
	}
	script := requireHTMLElement(t, document, "script", "")
	if source, _ := htmlAttribute(script, "src"); source != "/api/v4/earning-trends/assets/view.js" {
		t.Fatalf("script src = %q", source)
	}
	if _, ok := htmlAttribute(script, "defer"); !ok {
		t.Fatal("UI script is not deferred")
	}
	if elementsByTag(document, "style") != nil || strings.TrimSpace(htmlText(script)) != "" {
		t.Fatal("UI contains inline style or script content despite its strict CSP")
	}
	for _, id := range []string{"up-count", "stable-count", "down-count"} {
		requireHTMLElement(t, document, "", id)
	}
	if !strings.Contains(response.Body.String(), "Biggest demand") {
		t.Fatal("restored Biggest demand panel is missing")
	}
	if !strings.Contains(response.Body.String(), "network baseline") || strings.Contains(response.Body.String(), "previous period") {
		t.Fatal("UI does not explain the current network-baseline comparison")
	}
	mapPosition := strings.Index(response.Body.String(), `class="map-panel"`)
	listPosition := strings.Index(response.Body.String(), `class="country-panel"`)
	if mapPosition < 0 || listPosition < 0 || mapPosition >= listPosition {
		t.Fatal("restored layout must place the demand map before the country list")
	}

	search := requireHTMLElement(t, document, "input", "country-search")
	requireHTMLAttribute(t, search, "type", "search")
	label := requireHTMLElement(t, document, "label", "")
	requireHTMLAttribute(t, label, "for", "country-search")
	filters := elementsWithAttribute(document, "button", "data-filter")
	if len(filters) != 4 {
		t.Fatalf("trend filter buttons = %d, want 4", len(filters))
	}
	wantFilters := map[string]bool{"all": false, "increasing": false, "stable": false, "decreasing": false}
	selectedFilters := 0
	for _, button := range filters {
		requireHTMLAttribute(t, button, "type", "button")
		requireHTMLAttribute(t, button, "aria-controls", "country-list")
		pressed, ok := htmlAttribute(button, "aria-pressed")
		if !ok || (pressed != "true" && pressed != "false") {
			t.Fatalf("filter aria-pressed = %q, want true or false", pressed)
		}
		filter, _ := htmlAttribute(button, "data-filter")
		if _, ok := wantFilters[filter]; !ok {
			t.Fatalf("unexpected trend filter %q", filter)
		}
		wantFilters[filter] = true
		if pressed == "true" {
			selectedFilters++
			if filter != "increasing" {
				t.Fatalf("default filter = %q, want increasing", filter)
			}
		}
	}
	if selectedFilters != 1 {
		t.Fatalf("selected filters = %d, want 1", selectedFilters)
	}
	for filter, found := range wantFilters {
		if !found {
			t.Fatalf("missing trend filter %q", filter)
		}
	}

	resultCount := requireHTMLElement(t, document, "", "result-count")
	requireHTMLAttribute(t, resultCount, "role", "status")
	requireHTMLAttribute(t, resultCount, "aria-live", "polite")
	countryList := requireHTMLElement(t, document, "", "country-list")
	if _, noisy := htmlAttribute(countryList, "aria-live"); noisy {
		t.Fatal("the long country list must not be an aria-live region")
	}
	canvas := requireHTMLElement(t, document, "canvas", "world")
	requireHTMLAttribute(t, canvas, "tabindex", "0")
	requireHTMLAttribute(t, canvas, "aria-describedby", "map-help")
	if label, ok := htmlAttribute(canvas, "aria-label"); !ok || label == "" {
		t.Fatal("interactive map has no accessible label")
	}
	mapHelp := requireHTMLElement(t, document, "", "map-help")
	for _, instruction := range []string{"arrow keys", "Enter", "Escape", "country list"} {
		if !strings.Contains(htmlText(mapHelp), instruction) {
			t.Fatalf("map help does not explain %q", instruction)
		}
	}
	inspector := requireHTMLElement(t, document, "", "country-inspector")
	requireHTMLAttribute(t, inspector, "aria-live", "polite")
	mapStatus := requireHTMLElement(t, document, "", "map-status")
	requireHTMLAttribute(t, mapStatus, "role", "status")
	requireHTMLAttribute(t, mapStatus, "aria-live", "polite")
	for _, buttonID := range []string{"clear-search", "inspector-close", "zoom-in", "zoom-out", "reset-map"} {
		button := requireHTMLElement(t, document, "button", buttonID)
		requireHTMLAttribute(t, button, "type", "button")
	}

	for _, node := range allHTMLElements(document) {
		for _, attribute := range node.Attr {
			if strings.HasPrefix(strings.ToLower(attribute.Key), "on") || strings.EqualFold(attribute.Key, "style") {
				t.Fatalf("UI contains forbidden inline attribute %s on <%s>", attribute.Key, node.Data)
			}
		}
	}

	if got := response.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("cache control = %q, want no-cache", got)
	}
	if got := response.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := response.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("referrer policy = %q, want no-referrer", got)
	}
	csp := response.Header().Get("Content-Security-Policy")
	for _, directive := range []string{
		"default-src 'self'", "frame-ancestors 'none'", "object-src 'none'",
		"script-src 'self'", "style-src 'self'", "connect-src 'self'", "font-src 'self'",
	} {
		if !strings.Contains(csp, directive) {
			t.Fatalf("content security policy %q is missing %q", csp, directive)
		}
	}
	for _, forbidden := range []string{"'unsafe-inline'", "'unsafe-eval'", "http:", "https:", " *"} {
		if strings.Contains(csp, forbidden) {
			t.Fatalf("content security policy %q contains forbidden source %q", csp, forbidden)
		}
	}
}

func TestEarningTrendsStaticAssets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api := &APIByService{}
	tests := []struct {
		name         string
		path         string
		handler      gin.HandlerFunc
		contentType  string
		cacheControl string
	}{
		{
			name:         "stylesheet",
			path:         "/api/v4/earning-trends/assets/view.css",
			handler:      api.EarningTrendsCSS,
			contentType:  "text/css; charset=utf-8",
			cacheControl: "no-cache",
		},
		{
			name:         "javascript",
			path:         "/api/v4/earning-trends/assets/view.js",
			handler:      api.EarningTrendsJS,
			contentType:  "application/javascript; charset=utf-8",
			cacheControl: "no-cache",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			router.GET(test.path, test.handler)

			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", response.Code)
			}
			if got := response.Header().Get("Content-Type"); got != test.contentType {
				t.Fatalf("content type = %q, want %q", got, test.contentType)
			}
			if got := response.Header().Get("Cache-Control"); got != test.cacheControl {
				t.Fatalf("cache control = %q, want %q", got, test.cacheControl)
			}
			if got := response.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
			}
			etag := response.Header().Get("ETag")
			if etag == "" || etag != contentETag(response.Body.Bytes()) {
				t.Fatalf("ETag = %q, want hash of response body", etag)
			}
			if response.Body.Len() == 0 {
				t.Fatal("asset response is empty")
			}

			notModified := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			request.Header.Set("If-None-Match", etag)
			router.ServeHTTP(notModified, request)
			if notModified.Code != http.StatusNotModified {
				t.Fatalf("conditional status = %d, want 304", notModified.Code)
			}
			if notModified.Body.Len() != 0 {
				t.Fatalf("304 response contains %d body bytes", notModified.Body.Len())
			}
			if got := notModified.Header().Get("ETag"); got != etag {
				t.Fatalf("304 ETag = %q, want %q", got, etag)
			}
		})
	}
}

func TestEarningTrendsJavaScriptContracts(t *testing.T) {
	source := string(earningTrendsJS)

	t.Run("search and filters compose", func(t *testing.T) {
		requireSourceContains(t, source,
			"filter: 'increasing'",
			"function searchableText(value)",
			".normalize('NFD')",
			"function visibleCountries()",
			"searchableText(state.query.trim())",
			"state.filter === 'all' || country.trend === state.filter",
			"searchableText(country.country).includes(query)",
			"searchableText(country.country_code).includes(query)",
			"function setFilter(filter)",
			"function resetFilters()",
			"searchInput.addEventListener('input'",
		)
		if strings.Contains(source, "if (state.query && state.filter !== 'all') state.filter = 'all'") {
			t.Fatal("typing a search query resets the active trend filter instead of composing with it")
		}
	})

	t.Run("map and list share selection state", func(t *testing.T) {
		requireSourceContains(t, source,
			"selectedCode",
			"function selectCountry(code",
			"row.dataset.countryCode = country.country_code",
			"row.setAttribute('aria-current'",
			"selectCountry(country.country_code, { focusMap: true })",
			"selectCountry(item.code, { revealRow: true })",
			"visibleCountries().map(country => country.country_code)",
			"visibleCodes.has(item.code) ? fillColor(country) : mapColors.filtered",
			"if (event.key === 'Enter' || event.key === ' ')",
			"else if (event.key === 'Escape') clearSelection()",
		)
	})

	t.Run("map fills the available vertical space", func(t *testing.T) {
		requireSourceContains(t, source,
			"const mapHeight = Math.max(1, height - 24);",
		)
		if strings.Contains(source, "width / 2.5") {
			t.Fatal("map projection letterboxes the world instead of using the available canvas height")
		}
	})

	t.Run("failed demand data is not presented as stable", func(t *testing.T) {
		requireSourceContains(t, source,
			"dataLoading: false",
			"dataError: false",
			"unavailable:",
			"state.dataError = true",
			"if (state.dataError || !state.data) context.fillStyle = mapColors.unavailable",
			"Demand data unavailable",
			"function loadTrends()",
			"function loadMap()",
			"Retry demand data",
			"Retry map",
		)
	})

	t.Run("loading and stacked selection stay local", func(t *testing.T) {
		requireSourceContains(t, source,
			"if (state.dataLoading) return",
			"list.setAttribute('aria-busy', String(state.dataLoading))",
			"button.disabled = !available",
			"list.scrollTop",
		)
		if strings.Contains(source, "scrollIntoView") {
			t.Fatal("country selection can scroll the entire stacked page")
		}
	})

	t.Run("DOM rendering cannot expose arbitrary markup or pricing fields", func(t *testing.T) {
		requireSourceContains(t, source,
			"fetch('/api/v4/earning-trends?scope=countries'",
			"textContent",
			"replaceChildren",
			"const trend = trendClasses[country?.trend] ? country.trend : 'stable'",
			"const intensity = validIntensities.has(country?.intensity) ? country.intensity : 'low'",
		)
		for _, forbidden := range []string{
			"innerHTML", "outerHTML", "insertAdjacentHTML", "document.write", "eval(", "new Function(",
			"price_per_gib", "price_per_hour", "change_percentage", ".magnitude", ".service",
		} {
			if strings.Contains(source, forbidden) {
				t.Fatalf("JavaScript contains forbidden DOM or private-data pattern %q", forbidden)
			}
		}
	})
}

func TestEarningTrendsCSSAccessibilityAndResponsiveContracts(t *testing.T) {
	source := string(earningTrendsCSS)
	requireSourceContains(t, source,
		":focus-visible",
		"@media (max-width: 1080px)",
		"@media (max-width: 700px)",
		"@media (prefers-reduced-motion: reduce)",
		"touch-action: pan-y pinch-zoom",
		"width: 44px",
		"height: 44px",
	)

	t.Run("restored map and demand list stay bounded", func(t *testing.T) {
		requireSourceContains(t, source,
			".workspace { display: grid; height: 660px; grid-template-columns: minmax(0, 1.72fr) minmax(330px, .68fr);",
			".country-panel, .map-panel { width: 100%; min-width: 0; height: 100%; min-height: 0; overflow: hidden; }",
			".country-list { flex: 1; min-height: 0; overflow-y: auto;",
			".map-wrap { position: relative; flex: 1 1 auto; min-height: 0; overflow: hidden;",
		)
	})

	t.Run("stacked explorer panels keep explicit heights", func(t *testing.T) {
		requireSourceContains(t, source,
			"@media (max-width: 1200px)",
			".workspace { height: auto; grid-template-columns: 1fr; grid-template-rows: auto auto; }",
			".map-panel { height: 630px; }",
			".country-panel { height: 570px; }",
			".country-panel { height: 530px; }",
			".map-panel { height: 455px; }",
		)
	})
}

func TestEarningTrendsMap(t *testing.T) {
	gin.SetMode(gin.TestMode)
	api := &APIByService{}
	router := gin.New()
	router.GET("/api/v4/earning-trends/world.geojson", api.EarningTrendsMap)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v4/earning-trends/world.geojson", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Header().Get("Content-Type"), "application/geo+json") {
		t.Fatalf("map response: status=%d content-type=%q", response.Code, response.Header().Get("Content-Type"))
	}
	if got := response.Header().Get("Cache-Control"); got != "public, max-age=86400" {
		t.Fatalf("cache control = %q, want public, max-age=86400", got)
	}
	if got := response.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want nosniff", got)
	}
	etag := response.Header().Get("ETag")
	if etag == "" {
		t.Fatal("map response has no ETag")
	}

	var collection struct {
		Type     string `json:"type"`
		Features []struct {
			Properties struct {
				Code string `json:"ISO_A2_EH"`
				Name string `json:"NAME_LONG"`
			} `json:"properties"`
		} `json:"features"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &collection); err != nil {
		t.Fatalf("decode map: %v", err)
	}
	if collection.Type != "FeatureCollection" {
		t.Fatalf("map type = %q, want FeatureCollection", collection.Type)
	}

	supportedCodes := make(map[string]struct{}, len(supportedTrendCountries))
	for _, country := range supportedTrendCountries {
		supportedCodes[country.Code] = struct{}{}
	}
	mapCodes := make(map[string]struct{}, len(collection.Features))
	for _, feature := range collection.Features {
		if feature.Properties.Code == "AQ" || strings.EqualFold(feature.Properties.Name, "Antarctica") {
			t.Fatalf("map contains Antarctica feature: %+v", feature.Properties)
		}
		if len(feature.Properties.Code) == 2 && feature.Properties.Code != "-99" {
			mapCodes[feature.Properties.Code] = struct{}{}
			if _, ok := supportedCodes[feature.Properties.Code]; !ok {
				t.Fatalf("map contains country %s that pricing does not support", feature.Properties.Code)
			}
		}
	}
	if len(mapCodes) == 0 || len(mapCodes) >= len(supportedTrendCountries) {
		t.Fatalf("expected a non-empty low-resolution map subset, got %d map countries and %d supported countries", len(mapCodes), len(supportedTrendCountries))
	}

	notModified := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v4/earning-trends/world.geojson", nil)
	request.Header.Set("If-None-Match", etag)
	router.ServeHTTP(notModified, request)
	if notModified.Code != http.StatusNotModified {
		t.Fatalf("conditional map status = %d, want 304", notModified.Code)
	}
	if notModified.Body.Len() != 0 {
		t.Fatalf("304 map response contains %d body bytes", notModified.Body.Len())
	}
	if got := notModified.Header().Get("ETag"); got != etag {
		t.Fatalf("304 ETag = %q, want %q", got, etag)
	}
}

func requireHTMLElement(t *testing.T, root *html.Node, tag, id string) *html.Node {
	t.Helper()
	for _, element := range allHTMLElements(root) {
		if tag != "" && element.Data != tag {
			continue
		}
		if id != "" {
			value, ok := htmlAttribute(element, "id")
			if !ok || value != id {
				continue
			}
		}
		return element
	}
	if id != "" {
		t.Fatalf("UI is missing <%s id=%q>", tag, id)
	}
	t.Fatalf("UI is missing <%s>", tag)
	return nil
}

func requireHTMLAttribute(t *testing.T, node *html.Node, name, want string) {
	t.Helper()
	got, ok := htmlAttribute(node, name)
	if !ok || got != want {
		t.Fatalf("<%s> attribute %s = %q, want %q", node.Data, name, got, want)
	}
}

func htmlAttribute(node *html.Node, name string) (string, bool) {
	for _, attribute := range node.Attr {
		if strings.EqualFold(attribute.Key, name) {
			return attribute.Val, true
		}
	}
	return "", false
}

func allHTMLElements(root *html.Node) []*html.Node {
	var elements []*html.Node
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.ElementNode {
			elements = append(elements, node)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(root)
	return elements
}

func elementsByTag(root *html.Node, tag string) []*html.Node {
	var matches []*html.Node
	for _, element := range allHTMLElements(root) {
		if element.Data == tag {
			matches = append(matches, element)
		}
	}
	return matches
}

func elementsWithAttribute(root *html.Node, tag, attributeName string) []*html.Node {
	var matches []*html.Node
	for _, element := range allHTMLElements(root) {
		if element.Data != tag {
			continue
		}
		if _, ok := htmlAttribute(element, attributeName); ok {
			matches = append(matches, element)
		}
	}
	return matches
}

func htmlText(root *html.Node) string {
	var text strings.Builder
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Type == html.TextNode {
			text.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(root)
	return text.String()
}

func requireSourceContains(t *testing.T, source string, expected ...string) {
	t.Helper()
	for _, value := range expected {
		if !strings.Contains(source, value) {
			t.Errorf("source is missing required contract %q", value)
		}
	}
}

func history(current, previous *pricingbyservice.PriceByType) *pricingbyservice.PriceHistory {
	return &pricingbyservice.PriceHistory{Current: current, Previous: previous}
}

func servicePrices(wireguard, scraping, dvpn float64) *pricingbyservice.PriceByType {
	services := &pricingbyservice.PriceByServiceType{
		Wireguard: pricingbyservice.Price{PricePerGiBHumanReadable: wireguard},
		Scraping:  pricingbyservice.Price{PricePerGiBHumanReadable: scraping},
		DVPN:      pricingbyservice.Price{PricePerGiBHumanReadable: dvpn},
	}
	return &pricingbyservice.PriceByType{Residential: services}
}

func wireguardPrices(residentialGiB, residentialHour, otherGiB, otherHour float64) *pricingbyservice.PriceByType {
	return &pricingbyservice.PriceByType{
		Residential: &pricingbyservice.PriceByServiceType{Wireguard: pricingbyservice.Price{
			PricePerGiBHumanReadable:  residentialGiB,
			PricePerHourHumanReadable: residentialHour,
		}},
		Other: &pricingbyservice.PriceByServiceType{Wireguard: pricingbyservice.Price{
			PricePerGiBHumanReadable:  otherGiB,
			PricePerHourHumanReadable: otherHour,
		}},
	}
}

func uniformPrices(multiplier float64) *pricingbyservice.PriceByType {
	price := pricingbyservice.Price{
		PricePerGiBHumanReadable:  100 * multiplier,
		PricePerHourHumanReadable: 100 * multiplier,
	}
	services := func() *pricingbyservice.PriceByServiceType {
		return &pricingbyservice.PriceByServiceType{
			Wireguard:    price,
			Scraping:     price,
			QUICScraping: price,
			DataTransfer: price,
			DVPN:         price,
			Monitoring:   price,
			Runtime:      price,
		}
	}
	return &pricingbyservice.PriceByType{Residential: services(), Other: services()}
}
