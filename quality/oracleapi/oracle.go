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

func (a *API) Sessions() (*SessionsResponse, error) {
	resp, err := a.client.Get(fmt.Sprintf("%s/api/v1/providers/sessions", a.url))
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

type SessionsResponse struct {
	Connects    []Connect `json:"connects"`
	ConnectsMap map[string]*Connect
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
		return true
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
