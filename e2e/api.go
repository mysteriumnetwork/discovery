package e2e

import (
	"errors"
	"fmt"

	"github.com/dghubble/sling"

	"github.com/mysteriumnetwork/discovery/health"
	"github.com/mysteriumnetwork/discovery/price/pricingbyservice"
	v3 "github.com/mysteriumnetwork/discovery/proposal/v3"
)

var (
	DiscoveryAPI = newDiscoveryAPI(DiscoveryAPIurl)
	PricerAPI    = newPricingAPI(PricerAPIUrl)
)

func newDiscoveryAPI(basePath string) *discoveryAPI {
	return &discoveryAPI{
		basePath: basePath,
	}
}

type discoveryAPI struct {
	basePath string
}

func (a *discoveryAPI) ListFilters(query Query) (proposals []v3.Proposal, err error) {
	_, err = sling.New().Base(a.basePath).Get("/api/v3/proposals").QueryStruct(query).Receive(&proposals, nil)
	return proposals, err
}

func (a *discoveryAPI) GetStatus() (status health.StatusResponse, err error) {
	_, err = sling.New().Base(a.basePath).Get("/api/v3/status").Receive(&status, nil)
	return status, err
}

type Query struct {
	ProviderID              []string `url:"provider_id"`
	ServiceType             string   `url:"service_type"`
	Country                 string   `url:"location_country"`
	IPType                  string   `url:"ip_type"`
	AccessPolicy            string   `url:"access_policy"`
	AccessPolicySource      string   `url:"access_policy_source"`
	PriceGibMax             int64    `url:"price_gib_max"`
	PriceHourMax            int64    `url:"price_hour_max"`
	CompatibilityMin        int      `url:"compatibility_min"`
	CompatibilityMax        int      `url:"compatibility_max"`
	QualityMin              float64  `url:"quality_min"`
	IncludeMonitoringFailed bool     `url:"include_monitoring_failed"`
	NATCompatibility        string   `url:"nat_compatibility"`
}

func newPricingAPI(basePath string) *pricerAPI {
	return &pricerAPI{
		basePath: basePath,
	}
}

type pricerAPI struct {
	basePath string
}

func (a *pricerAPI) LatestPrices() (latestPrices pricingbyservice.LatestPrices, err error) {
	_, err = sling.New().Base(a.basePath).Get("/api/v4/prices").Receive(&latestPrices, nil)
	return latestPrices, err
}

func (a *pricerAPI) GetPriceConfig(token string) (config pricingbyservice.Config, err error) {
	resp, err := sling.New().Base(a.basePath).Add("Authorization", "Bearer "+token).Get("/api/v4/prices/config").Receive(&config, nil)
	if resp.StatusCode == 401 {
		return config, errors.New(fmt.Sprint(resp.StatusCode))
	}
	return config, err
}

func (a *pricerAPI) UpdatePriceConfig(token string, cfg pricingbyservice.Config) (err error) {
	resp, err := sling.New().Base(a.basePath).Add("Authorization", "Bearer "+token).BodyJSON(&cfg).Post("/api/v4/prices/config").Receive(nil, nil)
	if resp.StatusCode == 401 || resp.StatusCode == 400 {
		return errors.New(fmt.Sprint(resp.StatusCode))
	}
	return err
}
