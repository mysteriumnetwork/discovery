// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package oracleapi

import (
	"encoding/json"
	"fmt"
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
			Timeout: 5 * time.Second,
		},
	}
}

func (a *API) Quality(country string) (*ProposalQualityResponse, error) {
	resp, err := a.client.Get(fmt.Sprintf("%s/api/v2/providers/quality?country=%s", a.url, country))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("received response code: %v", resp.StatusCode)
	}

	var entries []ProposalQuality
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &ProposalQualityResponse{Entries: entries}, nil
}

type ProposalID struct {
	ProviderID  string `json:"providerId"`
	ServiceType string `json:"serviceType"`
}

type ProposalQuality struct {
	ProposalID ProposalID `json:"proposalId"`
	Quality    float32    `json:"quality"`
}

type ProposalQualityResponse struct {
	Entries []ProposalQuality
}

func (p ProposalQualityResponse) MarshalBinary() (data []byte, err error) {
	return json.Marshal(p)
}

func (p ProposalQualityResponse) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &p)
}
