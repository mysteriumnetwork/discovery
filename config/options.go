// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
)

type Options struct {
	DBHost           string
	DBPassword       string
	QualityOracleURL url.URL
	BrokerURL        url.URL
}

func Read() (*Options, error) {
	dbHost, err := requiredEnv("DB_HOST")
	if err != nil {
		return nil, err
	}
	dbPassword, err := requiredEnv("DB_PASSWORD")
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
	return &Options{
		DBHost:           dbHost,
		DBPassword:       dbPassword,
		QualityOracleURL: *qualityOracleURL,
		BrokerURL:        *brokerURL,
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
