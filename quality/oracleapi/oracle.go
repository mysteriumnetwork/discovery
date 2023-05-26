// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package oracleapi

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
	"time"
)

type API struct {
	client *http.Client
	url    string
}

func New(url string) *API {
	return &API{
		url: url,
		client: &http.Client{
			Timeout: 4 * time.Second,
		},
	}
}

func (a *API) Quality(country string) (map[string]*DetailedQuality, error) {
	resp, err := a.client.Get(fmt.Sprintf("%s/api/v2/providers/detailed?country=%s", a.url, country))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("received response code: %v", resp.StatusCode)
	}

	entries := make(map[string]*DetailedQuality)
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	log.Info().Msgf("Received %d entries", len(entries))
	log.Info().Msgf("Received %v", entries)

	return entries, nil
}

type ConnectCount struct {
	Success int `json:"success"`
	Fail    int `json:"fail"`
	Timeout int `json:"timeout"`
}

type DetailedQuality struct {
	Quality          float64 `json:"quality" example:"2.5"`
	MonitoringFailed bool    `json:"monitoringFailed,omitempty"`
	RestrictedNode   bool    `json:"restrictedNode,omitempty"`
	Latency          float64 `json:"latency" example:"75.5"`
	Uptime           float64 `json:"uptime" example:"7.7"`
	Bandwidth        float64 `json:"bandwidth" example:"15.5"`
}

// CountryLoad represents the ratio of providers to active sessions for country.
type CountryLoad struct {
	Providers uint64 `json:"providers"`
	Sessions  uint64 `json:"sessions"`
}

// NetworkLoadByCountry contains a map of country to relative load.
type NetworkLoadByCountry map[string]*CountryLoad
