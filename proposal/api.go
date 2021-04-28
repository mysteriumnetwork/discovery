// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package proposal

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/discovery/gorest"
	"github.com/rs/zerolog/log"
)

type API struct {
	service *Service
}

func NewAPI(service *Service) *API {
	return &API{service: service}
}

// Ping godoc.
// @Summary Ping
// @Description Ping
// @Accept json
// @Produce json
// @Success 200 {object} PingResponse
// @Router /ping [get]
func (a *API) Ping(c *gin.Context) {
	c.JSON(200, PingResponse{"pong"})
}

type PingResponse struct {
	Message string `json:"message"`
}

// Proposals list proposals.
// @Summary List proposals
// @Description List proposals
// @Param from query string false "Consumer country"
// @Param service_type query string false "Service type"
// @Param country query string false "Provider country"
// @Param residential query bool false "Residential IPs only?"
// @Param access_policy query string false "Access policy. When empty, returns only public proposals (default). Use * to return all."
// @Accept json
// @Product json
// @Success 200 {array} v2.Proposal
// @Router /proposals [get]
func (a *API) Proposals(c *gin.Context) {
	opts := ListOpts{
		from:         c.Query("from"),
		serviceType:  c.Query("service_type"),
		country:      c.Query("country"),
		accessPolicy: c.Query("access_policy"),
	}

	priceGiBMax, _ := strconv.ParseInt(c.Query("price_gib_max"), 10, 64)
	opts.priceGiBMax = priceGiBMax

	priceHourMax, _ := strconv.ParseInt(c.Query("price_hour_max"), 10, 64)
	opts.priceHourMax = priceHourMax

	compatibilityFrom, _ := strconv.ParseInt(c.Query("compatibility_from"), 10, 16)
	opts.compatibilityFrom = int(compatibilityFrom)

	compatibilityTo, _ := strconv.ParseInt(c.Query("compatibility_to"), 10, 16)
	opts.compatibilityTo = int(compatibilityTo)

	qlb, _ := strconv.ParseFloat(c.Query("quality_lower_bound"), 32)
	opts.qualityMin = float32(qlb)

	residential, _ := strconv.ParseBool(c.Query("residential"))
	opts.residential = residential

	proposals, err := a.service.List(opts)

	if err != nil {
		log.Err(err).Msg("Failed to list proposals")
		c.JSON(500, gorest.Err500)
		return
	}

	c.JSON(200, proposals)
}

func (a *API) RegisterRoutes(r gin.IRoutes) {
	r.GET("/ping", a.Ping)
	r.GET("/proposals", a.Proposals)
}
