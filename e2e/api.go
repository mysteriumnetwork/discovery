package e2e

import (
	"errors"
	"fmt"

	"github.com/dghubble/sling"
	"github.com/mysteriumnetwork/discovery/price/pricing"
	v3 "github.com/mysteriumnetwork/discovery/proposal/v3"
)

var discoveryAPI = newAPI(DiscoveryAPIurl)

func newAPI(basePath string) *api {
	return &api{
		basePath: basePath,
	}
}

type api struct {
	basePath string
}

func (a *api) LatestPrices() (latestPrices pricing.LatestPrices, err error) {
	_, err = sling.New().Base(a.basePath).Get("/api/v3/prices").Receive(&latestPrices, nil)
	return latestPrices, err
}

func (a *api) ListFilters(query Query) (proposals []v3.Proposal, err error) {
	_, err = sling.New().Base(a.basePath).Get("/api/v3/proposals").QueryStruct(query).Receive(&proposals, nil)
	return proposals, err
}

func (a *api) GetPriceConfig(token string) (config pricing.Config, err error) {
	resp, err := sling.New().Base(a.basePath).Add("Authorization", "Bearer "+token).Get("/api/v3/prices/config").Receive(&config, nil)
	if resp.StatusCode == 401 {
		return config, errors.New(fmt.Sprint(resp.StatusCode))
	}
	return config, err
}

func (a *api) UpdatePriceConfig(token string, cfg pricing.Config) (err error) {
	resp, err := sling.New().Base(a.basePath).Add("Authorization", "Bearer "+token).BodyJSON(&cfg).Post("/api/v3/prices/config").Receive(nil, nil)
	if resp.StatusCode == 401 || resp.StatusCode == 400 {
		return errors.New(fmt.Sprint(resp.StatusCode))
	}
	return err
}

type Query struct {
	From               string  `url:"from"`
	ProviderID         string  `url:"provider_id"`
	ServiceType        string  `url:"service_type"`
	Country            string  `url:"location_country"`
	IPType             string  `url:"ip_type"`
	AccessPolicy       string  `url:"access_policy"`
	AccessPolicySource string  `url:"access_policy_source"`
	PriceGibMax        int64   `url:"price_gib_max"`
	PriceHourMax       int64   `url:"price_hour_max"`
	CompatibilityMin   int     `url:"compatibility_min"`
	CompatibilityMax   int     `url:"compatibility_max"`
	QualityMin         float64 `url:"quality_min"`
	Tags               string  `url:"tags"`
}
