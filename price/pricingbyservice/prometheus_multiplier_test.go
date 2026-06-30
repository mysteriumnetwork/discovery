package pricingbyservice

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPrometheusDemandIndexProvider(t *testing.T) {
	transport := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		username, password, ok := r.BasicAuth()
		require.True(t, ok)
		require.Equal(t, "prom-user", username)
		require.Equal(t, "prom-password", password)
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/select/0/prometheus/api/v1/query", r.URL.Path)
		require.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		require.NoError(t, r.ParseForm())
		require.Equal(t, CountryDemandIndexQuery, r.Form.Get("query"))

		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body: io.NopCloser(strings.NewReader(`{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [
						{"metric": {"country": "US"}, "value": [123, "0.5"]},
						{"metric": {"country": "DE"}, "value": [123, "0.7"]},
						{"metric": {"country": "FR"}, "value": [123, "1"]},
						{"metric": {"country": "GB"}, "value": [123, "1.3"]},
						{"metric": {"country": "CA"}, "value": [123, "2"]},
						{"metric": {"country": "invalid"}, "value": [123, "0.5"]}
					]
				}
			}`)),
			Header: make(http.Header),
		}, nil
	})

	prometheusURL, err := url.Parse("https://prometheus.example/select/0/prometheus")
	require.NoError(t, err)
	provider := NewPrometheusDemandIndexProvider(prometheusURL, "prom-user", "prom-password", CountryDemandIndexQuery)
	provider.client = &http.Client{Transport: transport}

	got, err := provider.DemandIndexes(context.Background())
	require.NoError(t, err)
	require.Equal(t, map[ISO3166CountryCode]float64{
		"US": 0.5,
		"DE": 0.7,
		"FR": 1,
		"GB": 1.3,
		"CA": 2,
	}, got)
}

func TestPrometheusDemandIndexProviderWithoutAuthentication(t *testing.T) {
	transport := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		_, _, ok := r.BasicAuth()
		require.False(t, ok)
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body: io.NopCloser(strings.NewReader(`{
				"status": "success",
				"data": {"resultType": "vector", "result": []}
			}`)),
			Header: make(http.Header),
		}, nil
	})

	prometheusURL, err := url.Parse("https://prometheus.example")
	require.NoError(t, err)
	provider := NewPrometheusDemandIndexProvider(prometheusURL, "", "", CountryDemandIndexQuery)
	provider.client = &http.Client{Transport: transport}

	_, err = provider.DemandIndexes(context.Background())
	require.NoError(t, err)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestDailyCountryDemandIndexProvider(t *testing.T) {
	source := &stubCountryDemandIndexProvider{
		demandIndexes: map[ISO3166CountryCode]float64{"US": 0.05},
	}
	now := time.Date(2026, time.June, 18, 10, 0, 0, 0, time.FixedZone("UTC+6", 6*60*60))
	provider := NewDailyCountryDemandIndexProvider(source)
	provider.now = func() time.Time { return now }

	first, err := provider.DemandIndexes(context.Background())
	require.NoError(t, err)
	first["US"] = 0.1

	second, err := provider.DemandIndexes(context.Background())
	require.NoError(t, err)

	require.Equal(t, 1, source.calls)
	require.Equal(t, 0.05, second["US"])

	now = time.Date(2026, time.June, 19, 0, 0, 0, 0, time.UTC)
	_, err = provider.DemandIndexes(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, source.calls)
}

func TestNextUTCMidnight(t *testing.T) {
	now := time.Date(2026, time.June, 18, 23, 30, 0, 0, time.FixedZone("UTC+6", 6*60*60))
	require.Equal(
		t,
		time.Date(2026, time.June, 19, 0, 0, 0, 0, time.UTC),
		nextUTCMidnight(now),
	)
}

type stubCountryDemandIndexProvider struct {
	demandIndexes map[ISO3166CountryCode]float64
	calls         int
}

func (p *stubCountryDemandIndexProvider) DemandIndexes(context.Context) (map[ISO3166CountryCode]float64, error) {
	p.calls++
	return p.demandIndexes, nil
}

func TestGenerateNewPerCountryDefaultsMissingMultiplierToBalanced(t *testing.T) {
	price := PriceUSD{PricePerHour: 1, PricePerGiB: 2}
	cfg := Config{
		BasePrices: PriceByTypeUSD{
			Residential: &PriceByServiceTypeUSD{
				Wireguard: price, Scraping: price, QUICScraping: price,
				DataTransfer: price, DVPN: price, Monitoring: price,
			},
			Other: &PriceByServiceTypeUSD{
				Wireguard: price, Scraping: price, QUICScraping: price,
				DataTransfer: price, DVPN: price, Monitoring: price,
			},
		},
	}

	pricer := &PriceUpdater{}
	prices := pricer.generateNewPerCountryWithMultipliers(1, cfg, map[ISO3166CountryCode]float64{"US": 1.5})

	require.Equal(t, 1.5, prices["US"].Current.Residential.Wireguard.PricePerHourHumanReadable)
	require.Equal(t, 1.5, prices["US"].Current.Other.Wireguard.PricePerHourHumanReadable)
	require.Equal(t, float64(1), prices["DE"].Current.Residential.Wireguard.PricePerHourHumanReadable)
	require.Equal(t, float64(1), prices["DE"].Current.Other.Wireguard.PricePerHourHumanReadable)
}
