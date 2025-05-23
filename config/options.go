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
	LogLevel string

	QualityOracleURL url.URL
	QualityCacheTTL  time.Duration

	BrokerURL []url.URL

	RedisAddress []string
	RedisPass    string
	RedisDB      int

	UniverseJWTSecret string
	SentinelURL       string

	DevPass      string
	InternalPass string

	ProposalsHardLimitPerCountry int
	ProposalsSoftLimitPerCountry int

	CompatibilityMin int

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
	brokerURL, err := RequiredEnvURLs("BROKER_URL")
	if err != nil {
		return nil, err
	}

	compatibility, err := OptionalEnvInt("COMPATIBILITY_MIN", "0")
	if err != nil {
		return nil, err
	}

	devPass := OptionalEnv("DEV_PASS", "")
	internalPass := OptionalEnv("INTERNAL_PASS", "")
	logLevel := OptionalEnv("LOG_LEVEL", "debug")

	proposalsHardLimitPerCountry, err := OptionalEnvInt("PROPOSALS_HARD_LIMIT_PER_COUNTRY", "1000")
	if err != nil {
		return nil, err
	}
	proposalsSoftLimitPerCountry, err := OptionalEnvInt("PROPOSALS_SOFT_LIMIT_PER_COUNTRY", "1000")
	if err != nil {
		return nil, err
	}

	maxRequestsLimit := OptionalEnv("MAX_REQUESTS_LIMIT", "1000")
	limit, err := strconv.Atoi(maxRequestsLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to parse max requests limit: %w", err)
	}

	return &Options{
		QualityOracleURL:             *qualityOracleURL,
		QualityCacheTTL:              *qualityCacheTTL,
		BrokerURL:                    brokerURL,
		MaxRequestsLimit:             limit,
		DevPass:                      devPass,
		InternalPass:                 internalPass,
		ProposalsCacheTTL:            *proposalsCacheTTL,
		ProposalsCacheLimit:          proposalsCacheLimit,
		CountriesCacheLimit:          countriesCacheLimit,
		CompatibilityMin:             compatibility,
		LogLevel:                     logLevel,
		ProposalsHardLimitPerCountry: proposalsHardLimitPerCountry,
		ProposalsSoftLimitPerCountry: proposalsSoftLimitPerCountry,
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

	logLevel := OptionalEnv("LOG_LEVEL", "debug")

	return &Options{
		UniverseJWTSecret: universeJWTSecret,
		RedisAddress:      strings.Split(redisAddress, ";"),
		RedisPass:         redisPass,
		RedisDB:           redisDBint,
		SentinelURL:       sentinelURL,
		LogLevel:          logLevel,
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

func RequiredEnvURLs(key string) ([]url.URL, error) {
	strVal, err := RequiredEnv(key)
	if err != nil {
		return nil, err
	}

	parsedURLs := []url.URL{}

	for _, urlStr := range strings.Split(strVal, ";") {
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s from value '%s'", key, strVal)
		}

		parsedURLs = append(parsedURLs, *parsedURL)
	}

	if len(parsedURLs) == 0 {
		return nil, fmt.Errorf("no valid URLs found in %s", key)
	}

	return parsedURLs, nil
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
