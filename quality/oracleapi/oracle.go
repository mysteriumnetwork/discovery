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
			Timeout: 4 * time.Second,
		},
	}
}

func (a *API) NetworkLoad() (NetworkLoadByCountry, error) {
	resp, err := a.client.Get(fmt.Sprintf("%s/api/v2/countries/load", a.url))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("received response code: %v", resp.StatusCode)
	}

	var res NetworkLoadByCountry
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return res, err
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

func (a *API) Sessions(country string) (*SessionsResponse, error) {
	resp, err := a.client.Get(fmt.Sprintf("%s/api/v2/providers/sessions?country=%s", a.url, country))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("received response code: %v", resp.StatusCode)
	}

	var sp SessionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&sp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	sp.index()
	return &sp, nil
}

func (a *API) Latency(country string) (*LatencyResponse, error) {
	resp, err := a.client.Get(fmt.Sprintf("%s/api/v2/providers/latency?country=%s", a.url, country))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("received response code: %v", resp.StatusCode)
	}

	var lr LatencyResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr.Entries); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &lr, nil
}

func (a *API) Bandwidth(country string) (*BandwidthResponse, error) {
	resp, err := a.client.Get(fmt.Sprintf("%s/api/v2/providers/bandwidth?country=%s", a.url, country))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("received response code: %v", resp.StatusCode)
	}

	var br BandwidthResponse
	if err := json.NewDecoder(resp.Body).Decode(&br.Entries); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &br, nil
}

type BandwidthResponse struct {
	Entries []Bandwidth
}

func (p BandwidthResponse) MarshalBinary() (data []byte, err error) {
	return json.Marshal(p)
}

func (p BandwidthResponse) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &p)
}

type Bandwidth struct {
	ProposalID    ProposalID `json:"proposalId"`
	BandwidthMBPS float64    `json:"bandwidth"`
}

type LatencyResponse struct {
	Entries []Latency
}

func (p LatencyResponse) MarshalBinary() (data []byte, err error) {
	return json.Marshal(p)
}

func (p LatencyResponse) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &p)
}

type Latency struct {
	ProposalID ProposalID `json:"proposalId"`
	Latency    float64    `json:"latency"`
}

type SessionsResponse struct {
	Connects    []Connect `json:"connects"`
	ConnectsMap map[string]*Connect
}

func (p SessionsResponse) MarshalBinary() (data []byte, err error) {
	return json.Marshal(p)
}

func (p SessionsResponse) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &p)
}

func (s *SessionsResponse) index() {
	if s.ConnectsMap == nil {
		s.ConnectsMap = make(map[string]*Connect)
	}
	for i, c := range s.Connects {
		s.ConnectsMap[c.ProposalID.Key()] = &s.Connects[i]
	}
}

func (s *SessionsResponse) MonitoringFailed(providerID, serviceType string) bool {
	session, ok := s.ConnectsMap[serviceType+providerID]
	if !ok {
		return false
	}
	return session.MonitoringFailed
}

type Connect struct {
	ProposalID       ProposalID   `json:"proposalId"`
	ConnectCount     ConnectCount `json:"connectCount"`
	MonitoringFailed bool         `json:"monitoringFailed"`
}

type ConnectCount struct {
	Success int `json:"success"`
	Fail    int `json:"fail"`
	Timeout int `json:"timeout"`
}

type ProposalID struct {
	ProviderID  string `json:"providerId"`
	ServiceType string `json:"serviceType"`
}

func (pid *ProposalID) Key() string {
	return pid.ServiceType + pid.ProviderID
}

type ProposalQuality struct {
	ProposalID     ProposalID `json:"proposalId"`
	Quality        float64    `json:"quality"`
	RestrictedNode bool       `json:"restrictedNode"`
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

// CountryLoad represents the ratio of providers to active sessions for country.
type CountryLoad struct {
	Providers uint64 `json:"providers"`
	Sessions  uint64 `json:"sessions"`
}

// NetworkLoadByCountry contains a map of country to relative load.
type NetworkLoadByCountry map[string]*CountryLoad
