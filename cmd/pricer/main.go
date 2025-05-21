// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package main

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	_ "go.uber.org/automaxprocs"

	"github.com/mysteriumnetwork/discovery/config"
	_ "github.com/mysteriumnetwork/discovery/docs"
	"github.com/mysteriumnetwork/discovery/middleware"
	"github.com/mysteriumnetwork/discovery/price"
	"github.com/mysteriumnetwork/discovery/price/pricingbyservice"
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
	cfg, err := config.ReadPricer()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to read config")
	}

	mlog.SetLevel(logger, cfg.LogLevel)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger)
	r.Use(apierror.ErrorHandler)

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

	v4 := r.Group("/api/v4")

	cfger := pricingbyservice.NewConfigProviderDB(rdb)

	_, err = cfger.Get()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load cfg")
	}

	cfgerByService := pricingbyservice.NewConfigProviderDB(rdb)
	_, err = cfger.Get()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load cfg by service")
	}

	getterByService, err := pricingbyservice.NewPriceGetter(rdb)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize price getter by service")
	}

	ac := middleware.NewJWTChecker(cfg.SentinelURL, cfg.UniverseJWTSecret)
	price.NewAPIByService(rdb, getterByService, cfgerByService, ac).RegisterRoutes(v4)

	if err := r.Run(); err != nil {
		log.Err(err).Send()
		return
	}
}

func printBanner() {
	log.Info().Msg(strings.Repeat("▰", 60))
	log.Info().Msgf(" Starting discovery pricer version: %s", Version)
	log.Info().Msg(strings.Repeat("▰", 60))
}
