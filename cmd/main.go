// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package main

import (
	"context"
	stdlog "log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/mysteriumnetwork/discovery/price/pricing"

	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/discovery/config"
	"github.com/mysteriumnetwork/discovery/db"
	_ "github.com/mysteriumnetwork/discovery/docs"
	"github.com/mysteriumnetwork/discovery/listener"
	"github.com/mysteriumnetwork/discovery/price"
	"github.com/mysteriumnetwork/discovery/proposal"
	"github.com/mysteriumnetwork/discovery/quality"
	"github.com/mysteriumnetwork/discovery/quality/oracleapi"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
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

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	database := db.New(cfg.DbDSN)
	if err := database.Init(); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize DB")
	}
	defer database.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddress,
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

	proposalRepo := proposal.NewRepository(database)
	qualityOracleAPI := oracleapi.New(cfg.QualityOracleURL.String())
	qualityService := quality.NewService(qualityOracleAPI)
	proposalService := proposal.NewService(proposalRepo, qualityService)
	go proposalService.StartExpirationJob()
	defer proposalService.Shutdown()

	v3 := r.Group("/api/v3")
	proposal.NewAPI(proposalService, proposalRepo).RegisterRoutes(v3)

	cfger := pricing.NewConfigProviderDB(rdb)
	_, err = cfger.Get()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load cfg")
	}

	getter, err := pricing.NewPriceGetter(rdb)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize price getter")
	}

	price.NewAPI(getter, cfger, cfg.UniverseJWTSecret).RegisterRoutes(v3)

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
	writer := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02T15:04:05.000"}
	logger := log.Output(writer).Level(zerolog.DebugLevel).With().Caller().Timestamp().Logger()
	log.Logger = logger
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
