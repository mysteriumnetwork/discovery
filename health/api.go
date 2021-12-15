// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package health

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
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
	c.JSON(200, PingResponse{"pong"})
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := a.redis.Ping(ctx).Err()
	if err != nil {
		sr.CacheOK = false
		log.Err(err).Msg("could not reach redis")
	}

	c.JSON(200, sr)
}

type StatusResponse struct {
	CacheOK bool `json:"cache_ok"`
}

type API struct {
	redis *redis.Client
}

func NewAPI(redis *redis.Client) *API {
	return &API{
		redis: redis,
	}
}

func (a *API) RegisterRoutes(r gin.IRoutes) {
	r.GET("/ping", a.Ping)
	r.GET("/status", a.Status)
}
