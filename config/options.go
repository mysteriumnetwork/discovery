// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/mysteriumnetwork/payments/fees/price"
)

type Options struct {
	DbDSN                string
	QualityOracleURL     url.URL
	BrokerURL            url.URL
	GeckoURL             url.URL
	CoinRankingURL       url.URL
	CoinRankingToken     string
	UniverseJWTSecret    string
	DisablePricingUpdate bool
}

func Read() (*Options, error) {
	dsn, err := requiredEnv("DB_DSN")
	if err != nil {
		return nil, err
	}
	qualityOracleURL, err := requiredEnvURL("QUALITY_ORACLE_URL")
	if err != nil {
		return nil, err
	}
	brokerURL, err := requiredEnvURL("BROKER_URL")
	if err != nil {
		return nil, err
	}
	geckoURL, err := optionalEnvURL("GECKO_URL", price.DefaultGeckoURI)
	if err != nil {
		return nil, err
	}
	coinRankingURL, err := optionalEnvURL("COINRANKING_URL", price.DefaultCoinRankingURI)
	if err != nil {
		return nil, err
	}
	coinRankingToken, err := requiredEnv("COINRANKING_TOKEN")
	if err != nil {
		return nil, err
	}
	universeJWTSecret, err := requiredEnv("UNIVERSE_JWT_SECRET")
	if err != nil {
		return nil, err
	}
	disablePricingUpdate := optionalEnvBool("DISABLE_PRICING_UPDATE")
	return &Options{
		DbDSN:                dsn,
		QualityOracleURL:     *qualityOracleURL,
		BrokerURL:            *brokerURL,
		GeckoURL:             *geckoURL,
		CoinRankingURL:       *coinRankingURL,
		CoinRankingToken:     coinRankingToken,
		UniverseJWTSecret:    universeJWTSecret,
		DisablePricingUpdate: disablePricingUpdate,
	}, nil
}

func requiredEnv(key string) (string, error) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return "", fmt.Errorf("required environment variable is misssing: %s", key)
	}
	return val, nil
}

func requiredEnvURL(key string) (*url.URL, error) {
	strVal, err := requiredEnv(key)
	if err != nil {
		return nil, err
	}
	parsedURL, err := url.Parse(strVal)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s from value '%s'", key, strVal)
	}
	return parsedURL, nil
}

func optionalEnvURL(key string, defaults string) (*url.URL, error) {
	strVal := optionalEnv(key, defaults)
	parsedURL, err := url.Parse(strVal)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s from value '%s'", key, strVal)
	}
	return parsedURL, nil
}

func optionalEnv(key string, defaults string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return defaults
	}
	return val
}

func optionalEnvBool(key string) bool {
	val, _ := strconv.ParseBool(os.Getenv(key))
	return val
}
