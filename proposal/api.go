// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package proposal

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	cache "github.com/chenyahui/gin-cache"
	"github.com/chenyahui/gin-cache/persist"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type API struct {
	service           *Service
	location          locationProvider
	proposalsCache    *persist.MemoryStore
	countriesCache    *persist.MemoryStore
	proposalsCacheTTL time.Duration
}

type ctxCountryKey struct{}

type locationProvider interface {
	Country(ip string) (countryCode string, err error)
}

func NewAPI(service *Service,
	location locationProvider,
	proposalsCacheTTL time.Duration,
	proposalsCacheLimit int,
	countriesCacheLimit int,
) *API {
	a := &API{
		service:           service,
		location:          location,
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
	if from, ok := c.Request.Context().Value(ctxCountryKey{}).(string); ok {
		opts.from = from
	}

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
	if from, ok := c.Request.Context().Value(ctxCountryKey{}).(string); ok {
		opts.from = from
	}

	c.JSON(http.StatusOK, a.service.List(opts, false))
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
	if from, ok := c.Request.Context().Value(ctxCountryKey{}).(string); ok {
		opts.from = from
	}

	c.JSON(http.StatusOK, a.service.ListCountriesNumbers(opts, false))
}

func (a *API) RegisterRoutes(r gin.IRoutes) {
	countryMW := a.populateCountryMiddleware()
	cacheStrategy := a.newCacheStrategy()
	if a.proposalsCacheTTL > 0 {
		r.GET(
			"/countries",
			countryMW,
			cache.Cache(
				a.countriesCache,
				a.proposalsCacheTTL,
				cache.WithCacheStrategyByRequest(cacheStrategy),
			),
			a.CountriesNumbers,
		)
		r.GET(
			"/proposals",
			countryMW,
			cache.Cache(
				a.proposalsCache,
				a.proposalsCacheTTL,
				cache.WithCacheStrategyByRequest(cacheStrategy),
			),
			a.Proposals,
		)
	} else {
		r.GET("/countries", countryMW, a.CountriesNumbers)
		r.GET("/proposals", countryMW, a.Proposals)
	}
	r.GET("/proposals-metadata", a.ProposalsMetadata) // TODO move this into internal routes only once we migrate existing services to use it.
}

func (a *API) RegisterInternalRoutes(r gin.IRoutes) {
	countryMW := a.populateCountryMiddleware()
	cacheStrategy := a.newCacheStrategy()
	if a.proposalsCacheTTL > 0 {
		r.GET(
			"/proposals",
			countryMW,
			cache.Cache(
				a.proposalsCache,
				a.proposalsCacheTTL,
				cache.WithCacheStrategyByRequest(cacheStrategy),
			),
			a.AllProposals,
		)
	} else {
		r.GET("/proposals", countryMW, a.AllProposals)
	}
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
		from:               c.Query("from"),
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

func (a *API) populateCountryMiddleware() func(c *gin.Context) {
	return func(c *gin.Context) {
		from := c.Query("from")
		var err error
		if len(from) != 2 {
			from, err = a.location.Country(c.ClientIP())
		}
		if err != nil {
			from = "NL"
			log.Error().Err(err).Msg("Failed to autodetect client country")
		}
		c.Request = c.Request.WithContext(
			context.WithValue(c.Request.Context(), ctxCountryKey{}, from))
	}
}

func (a *API) newCacheStrategy() cache.GetCacheStrategyByRequest {
	return func(c *gin.Context) (bool, cache.Strategy) {
		from, _ := c.Request.Context().Value(ctxCountryKey{}).(string)
		newUri, err := getRequestUriIgnoreQueryOrder(
			c.Request.RequestURI,
			url.Values{
				"from": []string{from},
			},
		)
		if err != nil {
			newUri = c.Request.RequestURI
		}

		return true, cache.Strategy{
			CacheKey: newUri,
		}
	}
}

// from https://github.com/chenyahui/gin-cache/blob/cd1fa6cf7b54971a017034277c7e553f9f00ad02/cache.go#L166
func getRequestUriIgnoreQueryOrder(requestURI string, merge url.Values) (string, error) {
	parsedUrl, err := url.ParseRequestURI(requestURI)
	if err != nil {
		return "", err
	}

	values := parsedUrl.Query()
	for k, v := range merge {
		values[k] = v
	}

	// values.Encode will sort keys
	return parsedUrl.Path + "?" + values.Encode(), nil
}
