// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Ping godoc.
// @Summary Ping
// @Description Ping
// @Accept json
// @Produce json
// @Success 200 {object} PingResponse
// @Router /ping [get]
// @Tags system
func (a *API) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, PingResponse{"pong"})
}

type PingResponse struct {
	Message string `json:"message"`
}

// Status godoc.
// @Summary Status
// @Description Status
// @Accept json
// @Produce json
// @Success 200 {object} StatusResponse
// @Router /status [get]
// @Tags system
func (a *API) Status(c *gin.Context) {
	sr := StatusResponse{
		CacheOK: true,
	}

	c.JSON(http.StatusOK, sr)
}

type StatusResponse struct {
	CacheOK bool `json:"cache_ok"`
}

type API struct{}

func NewAPI() *API {
	return &API{}
}

func (a *API) RegisterRoutes(routers ...gin.IRoutes) {
	for _, r := range routers {
		r.GET("/ping", a.Ping)
		r.GET("/status", a.Status)
	}
}
