// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package proposal

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type API struct {
	service *Service
}

func NewAPI(service *Service) *API {
	return &API{service: service}
}

func (a *API) Ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}

func (a *API) Proposals(c *gin.Context) {
	proposals, err := a.service.List(ListOpts{
		from:        c.Query("from"),
		serviceType: c.Query("service_type"),
		country:     c.Query("country"),
	})

	if err != nil {
		log.Err(err).Msg("Failed to list proposals")
		c.JSON(500, "")
		return
	}

	c.JSON(200, proposals)
}

func (a *API) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/v3/ping", a.Ping)
	r.GET("/api/v3/proposals", a.Proposals)
}
