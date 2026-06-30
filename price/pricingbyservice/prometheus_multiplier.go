package pricingbyservice

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const CountryDemandIndexQuery = `WITH (
  requests = sum_over_time(dvpn_client_connect_requests_with_country{country!=""}[24h:10m]),
  VPN_DemandIndex =(0.4*sum by (country)(requests) / sum(requests) + 0.25*count by (country)(sum by (country,uuid)(requests)) / count(sum by (uuid)(requests))),

  Proxy_DemandIndex = (0.5*label_move(
    sum by (session_country)(sum_over_time(client_connection_data:sum10m{session_country!=""}[24h:10m]))
    /
    sum(sum_over_time(client_connection_data:sum10m{}[24h:10m]))
    ,"session_country","country")
    +
    0.3*sum by (country)(sum_over_time(proxy_total_requests:sum10m{country!=""}[24h:10m]))
    /
    sum(sum_over_time(proxy_total_requests:sum10m{}[24h:10m]))
    +
    0.2*label_move(
    count by (session_country)(sum by (session_country,uuid,sub_user_uuid)(sum_over_time(client_connection_data:sum10m{session_country!=""}[24h:10m])))
    /
    count(count by (uuid,sub_user_uuid)(sum_over_time(client_connection_data:sum10m{}[24h:10m])))
    ,"session_country","country")),

  Combined_Demand_Index = ((0.3*VPN_DemandIndex or on(country) (0 * Proxy_DemandIndex))
    +
    (0.7*Proxy_DemandIndex or on(country) (0 * VPN_DemandIndex))),


  Avg_provider_scoring = avg_over_time(provider_scoring{country!=""}[1h]),

  SupplyShare_country =  (
    ((count by (country)(Avg_provider_scoring >= 85) or 0*count by (country)(Avg_provider_scoring))
    +(0.75*count by (country)(Avg_provider_scoring >= 70 and Avg_provider_scoring < 85) or 0*count by (country)(Avg_provider_scoring))
    +(0.5*count by (country)(Avg_provider_scoring >= 55 and Avg_provider_scoring < 70) or 0*count by (country)(Avg_provider_scoring))
    +(0.2*count by (country)(Avg_provider_scoring >= 40 and Avg_provider_scoring < 55) or 0*count by (country)(Avg_provider_scoring))
    +(0*count by (country)(Avg_provider_scoring < 40)  or 0*count by (country)(Avg_provider_scoring)))
    /
    (count(Avg_provider_scoring >= 85)
    +0.75*count(Avg_provider_scoring >= 70 and Avg_provider_scoring < 85)
    +0.5*count (Avg_provider_scoring >= 55 and Avg_provider_scoring < 70)
    +0.2*count (Avg_provider_scoring >= 40 and Avg_provider_scoring < 55)
    +0*count (Avg_provider_scoring < 40)))
)

VPN_DemandIndex`

type CountryDemandIndexProvider interface {
	DemandIndexes(context.Context) (map[ISO3166CountryCode]float64, error)
}

type DailyCountryDemandIndexProvider struct {
	provider CountryDemandIndexProvider
	now      func() time.Time

	lock          sync.Mutex
	cached        map[ISO3166CountryCode]float64
	nextRefreshAt time.Time
}

func NewDailyCountryDemandIndexProvider(provider CountryDemandIndexProvider) *DailyCountryDemandIndexProvider {
	return &DailyCountryDemandIndexProvider{
		provider: provider,
		now:      time.Now,
	}
}

func (p *DailyCountryDemandIndexProvider) DemandIndexes(ctx context.Context) (map[ISO3166CountryCode]float64, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	now := p.now()
	if p.cached != nil && now.Before(p.nextRefreshAt) {
		return cloneFloatMap(p.cached), nil
	}

	demandIndexes, err := p.provider.DemandIndexes(ctx)
	if err != nil {
		return nil, err
	}

	p.cached = cloneFloatMap(demandIndexes)
	p.nextRefreshAt = nextUTCMidnight(now)
	return cloneFloatMap(p.cached), nil
}

func nextUTCMidnight(now time.Time) time.Time {
	utc := now.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day()+1, 0, 0, 0, 0, time.UTC)
}

func cloneFloatMap(source map[ISO3166CountryCode]float64) map[ISO3166CountryCode]float64 {
	result := make(map[ISO3166CountryCode]float64, len(source))
	for country, multiplier := range source {
		result[country] = multiplier
	}
	return result
}

type PrometheusDemandIndexProvider struct {
	baseURL  *url.URL
	username string
	password string
	query    string
	client   *http.Client
}

func NewPrometheusDemandIndexProvider(baseURL *url.URL, username, password, query string) *PrometheusDemandIndexProvider {
	return &PrometheusDemandIndexProvider{
		baseURL:  baseURL,
		username: username,
		password: password,
		query:    query,
		client:   http.DefaultClient,
	}
}

func (p *PrometheusDemandIndexProvider) DemandIndexes(ctx context.Context) (map[ISO3166CountryCode]float64, error) {
	endpoint := *p.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/api/v1/query"
	values := url.Values{}
	values.Set("query", p.query)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint.String(),
		strings.NewReader(values.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("create prometheus request: %w", err)
	}
	if p.username != "" || p.password != "" {
		req.SetBasicAuth(p.username, p.password)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query prometheus: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
		if readErr != nil {
			return nil, fmt.Errorf("query prometheus: unexpected status %s (read response: %v)", resp.Status, readErr)
		}
		return nil, fmt.Errorf("query prometheus: unexpected status %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var response prometheusQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode prometheus response: %w", err)
	}
	if response.Status != "success" {
		return nil, fmt.Errorf("query prometheus: status %q", response.Status)
	}
	if response.Data.ResultType != "vector" {
		return nil, fmt.Errorf("query prometheus: expected vector result, got %q", response.Data.ResultType)
	}

	demandIndexes := make(map[ISO3166CountryCode]float64, len(response.Data.Result))
	for _, result := range response.Data.Result {
		country := ISO3166CountryCode(result.Metric.Country)
		if country.Validate() != nil || len(result.Value) != 2 {
			continue
		}

		rawValue, ok := result.Value[1].(string)
		if !ok {
			continue
		}
		value, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			continue
		}
		if math.IsNaN(value) || math.IsInf(value, 0) {
			continue
		}
		demandIndexes[country] = value
	}

	return demandIndexes, nil
}

type prometheusQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
				Country string `json:"country"`
			} `json:"metric"`
			Value []any `json:"value"`
		} `json:"result"`
	} `json:"data"`
}
