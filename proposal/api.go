// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package proposal

import (
	"net/http"
	"strconv"
	"time"

	cache "github.com/chenyahui/gin-cache"
	"github.com/chenyahui/gin-cache/persist"
	"github.com/gin-gonic/gin"
)

type API struct {
	service           *Service
	proposalsCache    *persist.MemoryStore
	countriesCache    *persist.MemoryStore
	proposalsCacheTTL time.Duration
}

func NewAPI(service *Service,
	proposalsCacheTTL time.Duration,
	proposalsCacheLimit int,
	countriesCacheLimit int,
) *API {
	a := &API{
		service:           service,
		proposalsCacheTTL: proposalsCacheTTL,
	}
	if proposalsCacheTTL > 0 {
		a.proposalsCache = persist.NewMemoryStore(proposalsCacheTTL)
		a.proposalsCache.Cache.SetCacheSizeLimit(proposalsCacheLimit)
		a.countriesCache = persist.NewMemoryStore(proposalsCacheTTL)
		a.countriesCache.Cache.SetCacheSizeLimit(countriesCacheLimit)
	}
	return a
}

// Proposals list proposals.
// @Summary List proposals
// @Description List proposals
// @Param from query string false "Consumer country"
// @Param provider_id query string false "Provider ID"
// @Param service_type query string false "Service type"
// @Param location_country query string false "Provider country"
// @Param ip_type query string false "IP type (residential, datacenter, etc.)"
// @Param access_policy query string false "Access policy. When empty, returns only public proposals (default). Use 'all' to return all."
// @Param access_policy_source query string false "Access policy source"
// @Param compatibility_min query number false "Minimum compatibility. When empty, will not filter by it."
// @Param compatibility_max query number false "Maximum compatibility. When empty, will not filter by it."
// @Param quality_min query number false "Minimal quality threshold. When empty will be defaulted to 0. Quality ranges from [0.0; 3.0]"
// @Accept json
// @Product json
// @Success 200 {array} v3.Proposal
// @Router /proposals [get]
// @Tags proposals
func (a *API) Proposals(c *gin.Context) {
	opts := a.proposalArgs(c)

	c.JSON(http.StatusOK, a.service.List(opts, true))
}

// AllProposals list all proposals for internal use.
// @Summary List all proposals for internal use
// @Description List all proposals for internal use
// @Param from query string false "Consumer country"
// @Param provider_id query string false "Provider ID"
// @Param service_type query string false "Service type"
// @Param location_country query string false "Provider country"
// @Param ip_type query string false "IP type (residential, datacenter, etc.)"
// @Param access_policy query string false "Access policy. When empty, returns only public proposals (default). Use 'all' to return all."
// @Param access_policy_source query string false "Access policy source"
// @Param compatibility_min query number false "Minimum compatibility. When empty, will not filter by it."
// @Param compatibility_max query number false "Maximum compatibility. When empty, will not filter by it."
// @Param quality_min query number false "Minimal quality threshold. When empty will be defaulted to 0. Quality ranges from [0.0; 3.0]"
// @Accept json
// @Product json
// @Success 200 {array} v3.Proposal
// @Router /proposals [get]
// @Tags proposals
func (a *API) AllProposals(c *gin.Context) {
	opts := a.proposalArgs(c)

	c.JSON(http.StatusOK, a.service.List(opts, false))
}

// AggregatedProposals list aggregated proposals.
// @Summary List aggregated proposals
// @Description List aggregated proposals
// @Param from query string false "Consumer country"
// @Param provider_id query string false "Provider ID"
// @Param service_type query string false "Service type"
// @Param location_country query string false "Provider country"
// @Param ip_type query string false "IP type (residential, datacenter, etc.)"
// @Param access_policy query string false "Access policy. When empty, returns only public proposals (default). Use 'all' to return all."
// @Param access_policy_source query string false "Access policy source"
// @Param compatibility_min query number false "Minimum compatibility. When empty, will not filter by it."
// @Param compatibility_max query number false "Maximum compatibility. When empty, will not filter by it."
// @Param quality_min query number false "Minimal quality threshold. When empty will be defaulted to 0. Quality ranges from [0.0; 3.0]"
//
// @Accept json
// @Product json
// @Success 200 {array} v3.Proposal
// @Router /proposals/aggregated [get]
// @Tags proposals
func (a *API) AggregatedProposals(c *gin.Context) {
	opts := a.proposalArgs(c)

	c.JSON(http.StatusOK, a.service.ListAggregated(opts))
}

// CountriesNumbers list number of providers in each country.
// @Summary List number of providers in each country
// @Description List number of providers in each country
// @Param from query string false "Consumer country"
// @Param provider_id query string false "Provider ID"
// @Param service_type query string false "Service type"
// @Param location_country query string false "Provider country"
// @Param ip_type query string false "IP type (residential, datacenter, etc.)"
// @Param access_policy query string false "Access policy. When empty, returns only public proposals (default). Use 'all' to return all."
// @Param access_policy_source query string false "Access policy source"
// @Param compatibility_min query number false "Minimum compatibility. When empty, will not filter by it."
// @Param compatibility_max query number false "Maximum compatibility. When empty, will not filter by it."
// @Param quality_min query number false "Minimal quality threshold. When empty will be defaulted to 0. Quality ranges from [0.0; 3.0]"
// @Accept json
// @Product json
// @Router /countries [get]
// @Tags countries
func (a *API) CountriesNumbers(c *gin.Context) {
	opts := a.proposalArgs(c)

	c.JSON(http.StatusOK, a.service.ListCountriesNumbers(opts, false))
}

func (a *API) RegisterRoutes(r gin.IRoutes) {
	cacheStrategy := a.newCacheStrategy()
	if a.proposalsCacheTTL > 0 {
		r.GET(
			"/countries",
			cache.Cache(
				a.countriesCache,
				a.proposalsCacheTTL,
				cache.WithCacheStrategyByRequest(cacheStrategy),
			),
			a.CountriesNumbers,
		)
		r.GET(
			"/proposals",
			cache.Cache(
				a.proposalsCache,
				a.proposalsCacheTTL,
				cache.WithCacheStrategyByRequest(cacheStrategy),
			),
			a.Proposals,
		)
	} else {
		r.GET("/countries", a.CountriesNumbers)
		r.GET("/proposals", a.Proposals)
	}
	r.GET("/proposals-metadata", a.ProposalsMetadata) // TODO move this into internal routes only once we migrate existing services to use it.
}

func (a *API) RegisterInternalRoutes(r gin.IRoutes) {
	cacheStrategy := a.newCacheStrategy()
	if a.proposalsCacheTTL > 0 {
		r.GET(
			"/proposals",
			cache.Cache(
				a.proposalsCache,
				a.proposalsCacheTTL,
				cache.WithCacheStrategyByRequest(cacheStrategy),
			),
			a.AllProposals,
		)
	} else {
		r.GET("/proposals", a.AllProposals)
	}
	r.GET("/proposals/aggregated", a.AggregatedProposals)
	r.GET("/proposals-metadata", a.ProposalsMetadata)
}

// ProposalsMetadata list proposals' metadata.
// @Summary List proposals' metadata.
// @Description List proposals' metadata
// @Param provider_id query string false "Provider ID"
// @Accept json
// @Product json
// @Success 200 {array} v3.Metadata
// @Router /proposals-metadata [get]
func (a *API) ProposalsMetadata(c *gin.Context) {
	opts := repoMetadataOpts{
		providerID: c.Query("provider_id"),
	}
	metadata := a.service.Metadata(opts)
	c.JSON(http.StatusOK, metadata)
}

func (a *API) proposalArgs(c *gin.Context) ListOpts {
	opts := ListOpts{
		serviceType:        c.Query("service_type"),
		locationCountry:    c.Query("location_country"),
		accessPolicy:       c.Query("access_policy"),
		accessPolicySource: c.Query("access_policy_source"),
		ipType:             c.Query("ip_type"),
	}

	pids, _ := c.GetQueryArray("provider_id")
	opts.providerIDS = pids

	compatibilityMin, _ := strconv.ParseInt(c.Query("compatibility_min"), 10, 16)
	opts.compatibilityMin = int(compatibilityMin)

	compatibilityMax, _ := strconv.ParseInt(c.Query("compatibility_max"), 10, 16)
	opts.compatibilityMax = int(compatibilityMax)

	bandwidthMin, _ := strconv.ParseFloat(c.Query("bandwidth_min"), 16)
	opts.bandwidthMin = bandwidthMin

	qualityMin, _ := strconv.ParseFloat(c.Query("quality_min"), 64)
	opts.qualityMin = qualityMin

	includeMonitoringFailed, _ := strconv.ParseBool(c.Query("include_monitoring_failed"))
	opts.includeMonitoringFailed = includeMonitoringFailed

	natCompatibility := c.Query("nat_compatibility")
	opts.natCompatibility = natCompatibility

	presetID, _ := strconv.ParseInt(c.Query("preset_id"), 10, 16)
	opts.presetID = int(presetID)

	return opts
}

func (a *API) newCacheStrategy() cache.GetCacheStrategyByRequest {
	return func(c *gin.Context) (bool, cache.Strategy) {
		return true, cache.Strategy{
			CacheKey: c.Request.RequestURI,
		}
	}
}
