package e2e

import (
	"github.com/dghubble/sling"
	v2 "github.com/mysteriumnetwork/discovery/proposal/v2"
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

func (a *api) ListFilters(query Query) (proposals []v2.Proposal, err error) {
	_, err = sling.New().Base(a.basePath).Get("/api/v3/proposals").QueryStruct(query).Receive(&proposals, nil)
	return proposals, err
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
}
