// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Options struct {
	QualityOracleURL url.URL
	QualityCacheTTL  time.Duration

	BrokerURL url.URL

	RedisAddress []string
	RedisPass    string
	RedisDB      int

	BadgerAddress url.URL

	LocationAddress url.URL
	LocationUser    string
	LocationPass    string

	UniverseJWTSecret string
	SentinelURL       string

	DevPass string

	MaxRequestsLimit int

	ProposalsCacheTTL   time.Duration
	ProposalsCacheLimit int
	CountriesCacheLimit int
}

func ReadDiscovery() (*Options, error) {
	qualityOracleURL, err := RequiredEnvURL("QUALITY_ORACLE_URL")
	if err != nil {
		return nil, err
	}
	qualityCacheTTL, err := RequiredEnvDuration("QUALITY_CACHE_TTL")
	if err != nil {
		return nil, err
	}
	proposalsCacheTTL, err := OptionalEnvDuration("PROPOSALS_CACHE_TTL", "1m")
	if err != nil {
		return nil, err
	}
	proposalsCacheLimit, err := OptionalEnvInt("PROPOSALS_CACHE_LIMIT", "100")
	if err != nil {
		return nil, err
	}
	countriesCacheLimit, err := OptionalEnvInt("COUNTRIES_CACHE_LIMIT", "1000")
	if err != nil {
		return nil, err
	}
	brokerURL, err := RequiredEnvURL("BROKER_URL")
	if err != nil {
		return nil, err
	}
	badgerAddress, err := RequiredEnvURL("BADGER_ADDRESS")
	if err != nil {
		return nil, err
	}

	locationUser := OptionalEnv("LOCATION_USER", "")
	locationPass := OptionalEnv("LOCATION_PASS", "")
	locationAddress, err := RequiredEnvURL("LOCATION_ADDRESS")
	if err != nil {
		return nil, err
	}

	devPass := OptionalEnv("DEV_PASS", "")

	maxRequestsLimit := OptionalEnv("MAX_REQUESTS_LIMIT", "1000")
	limit, err := strconv.Atoi(maxRequestsLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to parse max requests limit: %w", err)
	}

	return &Options{
		QualityOracleURL:    *qualityOracleURL,
		QualityCacheTTL:     *qualityCacheTTL,
		BrokerURL:           *brokerURL,
		BadgerAddress:       *badgerAddress,
		LocationAddress:     *locationAddress,
		LocationUser:        locationUser,
		LocationPass:        locationPass,
		MaxRequestsLimit:    limit,
		DevPass:             devPass,
		ProposalsCacheTTL:   *proposalsCacheTTL,
		ProposalsCacheLimit: proposalsCacheLimit,
		CountriesCacheLimit: countriesCacheLimit,
	}, nil
}

func ReadPricer() (*Options, error) {
	universeJWTSecret, err := RequiredEnv("UNIVERSE_JWT_SECRET")
	if err != nil {
		return nil, err
	}
	redisAddress, err := RequiredEnv("REDIS_ADDRESS")
	if err != nil {
		return nil, err
	}

	sentinelURL, err := RequiredEnv("SENTINEL_URL")
	if err != nil {
		return nil, err
	}

	redisPass := OptionalEnv("REDIS_PASS", "")

	redisDBint := 0
	redisDB := OptionalEnv("REDIS_DB", "0")
	if redisDB != "" {
		res, err := strconv.Atoi(redisDB)
		if err != nil {
			return nil, fmt.Errorf("could not parse redis db from %q: %w", redisDB, err)
		}
		redisDBint = res
	}

	return &Options{
		UniverseJWTSecret: universeJWTSecret,
		RedisAddress:      strings.Split(redisAddress, ";"),
		RedisPass:         redisPass,
		RedisDB:           redisDBint,
		SentinelURL:       sentinelURL,
	}, nil
}

func RequiredEnv(key string) (string, error) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return "", fmt.Errorf("required environment variable is missing: %s", key)
	}
	return val, nil
}

func RequiredEnvURL(key string) (*url.URL, error) {
	strVal, err := RequiredEnv(key)
	if err != nil {
		return nil, err
	}
	parsedURL, err := url.Parse(strVal)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s from value '%s'", key, strVal)
	}
	return parsedURL, nil
}

func RequiredEnvDuration(key string) (*time.Duration, error) {
	strVal, err := RequiredEnv(key)
	if err != nil {
		return nil, err
	}

	duration, err := time.ParseDuration(strVal)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s from value '%s'", key, strVal)
	}

	return &duration, nil
}

func OptionalEnvDuration(key string, defaults string) (*time.Duration, error) {
	strVal := OptionalEnv(key, defaults)
	duration, err := time.ParseDuration(strVal)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s from value '%s'", key, strVal)
	}

	return &duration, nil
}

func OptionalEnvInt(key string, defaults string) (int, error) {
	strVal := OptionalEnv(key, defaults)
	intVal, err := strconv.ParseInt(strVal, 10, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to parse %s from value '%s'", key, strVal)
	}

	return int(intVal), nil
}

func OptionalEnvURL(key string, defaults string) (*url.URL, error) {
	strVal := OptionalEnv(key, defaults)
	parsedURL, err := url.Parse(strVal)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s from value '%s'", key, strVal)
	}
	return parsedURL, nil
}

func OptionalEnv(key string, defaults string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return defaults
	}
	return val
}

func OptionalEnvBool(key string) bool {
	val, _ := strconv.ParseBool(os.Getenv(key))
	return val
}
