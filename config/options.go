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
	DBConnString     string
	QualityOracleURL url.URL
	BrokerURL        url.URL
	BindAddr         string
}

func Read() (*Options, error) {
	dbConnString, err := requiredEnv("DB_CONN_STRING")
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
		DBConnString:     dbConnString,
		QualityOracleURL: *qualityOracleURL,
		BrokerURL:        *brokerURL,
		BindAddr:         optionalEnv("BIND_ADDR", ":8080"),
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
