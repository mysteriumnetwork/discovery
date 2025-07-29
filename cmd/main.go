// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "go.uber.org/automaxprocs"

	"github.com/mysteriumnetwork/discovery/config"
	_ "github.com/mysteriumnetwork/discovery/docs"
	"github.com/mysteriumnetwork/discovery/health"
	"github.com/mysteriumnetwork/discovery/listener"
	"github.com/mysteriumnetwork/discovery/middleware"
	"github.com/mysteriumnetwork/discovery/proposal"
	"github.com/mysteriumnetwork/discovery/quality"
	"github.com/mysteriumnetwork/discovery/quality/oracleapi"
	"github.com/mysteriumnetwork/go-rest/apierror"
	mlog "github.com/mysteriumnetwork/logger"
)

var Version = "<dev>"

// @title Discovery API
// @version 3.0
// @BasePath /api/v3
// @description Discovery API for Mysterium Network
func main() {
	logger := mlog.BootstrapDefaultLogger()
	printBanner()
	cfg, err := config.ReadDiscovery()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to read config")
	}
	mlog.SetLevel(logger, cfg.LogLevel)

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

	proposalRepo := proposal.NewRepository(cfg.ProposalsHardLimitPerCountry, cfg.ProposalsSoftLimitPerCountry, cfg.CompatibilityMin)
	qualityOracleAPI := oracleapi.New(cfg.QualityOracleURL.String())
	qualityService := quality.NewService(qualityOracleAPI, cfg.QualityCacheTTL)
	proposalService := proposal.NewService(proposalRepo, qualityService)
	go proposalService.StartExpirationJob()
	defer proposalService.Shutdown()

	limitMW := LimitMiddleware(cfg.MaxRequestsLimit)
	v3 := r.Group("/api/v3")
	v3.Use(limitMW)
	v4 := r.Group("/api/v4")
	v4.Use(limitMW)

	internal := r.Group("/internal/v4", gin.BasicAuth(gin.Accounts{
		"internal": cfg.InternalPass,
	}))
	internal.Use(limitMW)

	proposalsAPI := proposal.NewAPI(
		proposalService,
		cfg.ProposalsCacheTTL,
		cfg.ProposalsCacheLimit,
		cfg.CountriesCacheLimit,
	)
	proposalsAPI.RegisterRoutes(v3)
	proposalsAPI.RegisterRoutes(v4)
	proposalsAPI.RegisterInternalRoutes(internal)

	health.NewAPI().RegisterRoutes(v3, v4, internal)

	for _, brokerURL := range cfg.BrokerURL {
		brokerListener := connectToBroker(brokerURL.String(), proposalRepo)
		if brokerListener != nil {
			defer brokerListener.Shutdown()
		} else {
			log.Fatal().Msgf("Failed to connect to broker %s, stopping", brokerURL)
		}
	}

	if err := r.Run(); err != nil {
		log.Err(err).Send()
		return
	}
}

func connectToBroker(brokerURL string, proposalRepo *proposal.Repository) *listener.Listener {
	brokerListener := listener.New(brokerURL, proposalRepo)

	const maxRetries = 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := brokerListener.Listen(); err != nil {
			if attempt == maxRetries-1 {
				log.Error().Err(err).Msgf("Could not connect to broker %s after %d attempts, skipping", brokerURL, maxRetries)

				return nil
			}

			// Exponential backoff: 1s, 2s, 4s
			backoffDuration := time.Duration(1<<attempt) * time.Second
			log.Warn().Err(err).Msgf("Failed to connect to broker %s (attempt %d/%d), retrying in %v...",
				brokerURL, attempt+1, maxRetries, backoffDuration)
			time.Sleep(backoffDuration)
		} else {
			log.Info().Msgf("Successfully connected to broker %s", brokerURL)

			return brokerListener
		}
	}

	return nil
}

func printBanner() {
	log.Info().Msg(strings.Repeat("▰", 60))
	log.Info().Msgf(" Starting discovery version: %s", Version)
	log.Info().Msg(strings.Repeat("▰", 60))
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
