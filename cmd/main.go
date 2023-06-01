// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package main

import (
	stdlog "log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "go.uber.org/automaxprocs"

	"github.com/gin-contrib/pprof"
	"github.com/mysteriumnetwork/discovery/config"
	_ "github.com/mysteriumnetwork/discovery/docs"
	"github.com/mysteriumnetwork/discovery/health"
	"github.com/mysteriumnetwork/discovery/listener"
	"github.com/mysteriumnetwork/discovery/location"
	"github.com/mysteriumnetwork/discovery/middleware"
	"github.com/mysteriumnetwork/discovery/proposal"
	"github.com/mysteriumnetwork/discovery/quality"
	"github.com/mysteriumnetwork/discovery/quality/oracleapi"
	"github.com/mysteriumnetwork/discovery/tags"
	"github.com/mysteriumnetwork/go-rest/apierror"
	mlog "github.com/mysteriumnetwork/logger"
)

var Version = "<dev>"

// @title Discovery API
// @version 3.0
// @BasePath /api/v3
// @description Discovery API for Mysterium Network
func main() {
	configureLogger()
	printBanner()
	cfg, err := config.ReadDiscovery()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to read config")
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger)
	r.Use(apierror.ErrorHandler)

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	if cfg.DevPass != "" {
		devGroup := r.Group("/dev", gin.BasicAuth(gin.Accounts{
			"dev": cfg.DevPass,
		}))
		pprof.RouteRegister(devGroup, "pprof")
	}

	tagEnhancer := tags.NewEnhancer(tags.NewApi(cfg.BadgerAddress.String()))
	proposalRepo := proposal.NewRepository([]proposal.Enhancer{tagEnhancer})
	qualityOracleAPI := oracleapi.New(cfg.QualityOracleURL.String())
	qualityService := quality.NewService(qualityOracleAPI, cfg.QualityCacheTTL)
	proposalService := proposal.NewService(proposalRepo, qualityService)
	go proposalService.StartExpirationJob()
	defer proposalService.Shutdown()

	locationProvider := location.NewLocationProvider(cfg.LocationAddress.String(), cfg.LocationUser, cfg.LocationPass)

	limitMW := LimitMiddleware(cfg.MaxRequestsLimit)
	v3 := r.Group("/api/v3")
	v3.Use(limitMW)
	v4 := r.Group("/api/v4")
	v4.Use(limitMW)

	proposalsAPI := proposal.NewAPI(
		proposalService,
		locationProvider,
		cfg.ProposalsCacheTTL,
		cfg.ProposalsCacheLimit,
		cfg.CountriesCacheLimit,
	)
	proposalsAPI.RegisterRoutes(v3)
	proposalsAPI.RegisterRoutes(v4)

	health.NewAPI().RegisterRoutes(v3)
	health.NewAPI().RegisterRoutes(v4)

	brokerListener := listener.New(cfg.BrokerURL.String(), proposalRepo)

	if err := brokerListener.Listen(); err != nil {
		log.Fatal().Err(err).Msg("Could not listen to the broker")
	}
	defer brokerListener.Shutdown()

	if err := r.Run(); err != nil {
		log.Err(err).Send()
		return
	}
}

func configureLogger() {
	mlog.BootstrapDefaultLogger()
	stdlog.SetFlags(0)
	stdlog.SetOutput(log.Logger)
}

func printBanner() {
	log.Info().Msg(strings.Repeat("▰", 60))
	log.Info().Msgf(" Starting discovery version: %s", Version)
	log.Info().Msg(strings.Repeat("▰", 60))
	log.Info().Msg(" She has carried us into the future")
	log.Info().Msg(" and it will be our privilege to make that future bright.")
	log.Info().Msg(strings.Repeat("▱", 60))
}

func LimitMiddleware(size int) gin.HandlerFunc {
	limit := make(chan struct{}, size)
	return func(c *gin.Context) {
		select {
		case limit <- struct{}{}:
			c.Next()
			<-limit
		default:
			c.AbortWithStatus(http.StatusTooManyRequests)
		}
	}
}
