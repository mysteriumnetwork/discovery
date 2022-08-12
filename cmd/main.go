// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package main

import (
	"context"
	stdlog "log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "go.uber.org/automaxprocs"

	"github.com/mysteriumnetwork/discovery/config"
	_ "github.com/mysteriumnetwork/discovery/docs"
	"github.com/mysteriumnetwork/discovery/health"
	"github.com/mysteriumnetwork/discovery/listener"
	"github.com/mysteriumnetwork/discovery/location"
	"github.com/mysteriumnetwork/discovery/price"
	"github.com/mysteriumnetwork/discovery/price/pricing"
	"github.com/mysteriumnetwork/discovery/price/pricingbyservice"
	"github.com/mysteriumnetwork/discovery/proposal"
	"github.com/mysteriumnetwork/discovery/quality"
	"github.com/mysteriumnetwork/discovery/quality/oracleapi"
	"github.com/mysteriumnetwork/discovery/tags"
	"github.com/mysteriumnetwork/discovery/token"
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
	cfg, err := config.Read()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to read config")
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(mlog.GinLogFunc())
	r.Use(apierror.ErrorHandler)

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    cfg.RedisAddress,
		Password: cfg.RedisPass,
		DB:       cfg.RedisDB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	st := rdb.Ping(ctx)
	err = st.Err()
	if err != nil {
		log.Fatal().Err(err).Msg("could not reach redis")
	}

	tagEnhancer := tags.NewEnhancer(tags.NewApi(cfg.BadgerAddress.String()))
	proposalRepo := proposal.NewRepository([]proposal.Enhancer{tagEnhancer})
	qualityOracleAPI := oracleapi.New(cfg.QualityOracleURL.String())
	qualityService := quality.NewService(qualityOracleAPI, cfg.QualityCacheTTL)
	proposalService := proposal.NewService(proposalRepo, qualityService)
	go proposalService.StartExpirationJob()
	defer proposalService.Shutdown()

	locationProvider := location.NewLocationProvider(cfg.LocationAddress.String(), cfg.LocationUser, cfg.LocationPass)

	v3 := r.Group("/api/v3")
	v4 := r.Group("/api/v4")

	proposal.NewAPI(proposalService, proposalRepo, locationProvider).RegisterRoutes(v3)
	proposal.NewAPI(proposalService, proposalRepo, locationProvider).RegisterRoutes(v4)

	health.NewAPI(rdb).RegisterRoutes(v3)
	health.NewAPI(rdb).RegisterRoutes(v4)

	cfger := pricing.NewConfigProviderDB(rdb)
	_, err = cfger.Get()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load cfg")
	}
	cfgerByService := pricingbyservice.NewConfigProviderDB(rdb)
	_, err = cfger.Get()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load cfg by service")
	}

	getter, err := pricing.NewPriceGetter(rdb)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize price getter")
	}
	getterByService, err := pricingbyservice.NewPriceGetter(rdb)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize price getter by service")
	}

	ac := token.NewJWTChecker(cfg.SentinelURL, cfg.UniverseJWTSecret)
	price.NewAPI(getter, cfger, ac).RegisterRoutes(v3)
	price.NewAPIByService(getterByService, cfgerByService, ac).RegisterRoutes(v4)

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
