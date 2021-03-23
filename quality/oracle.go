package quality

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
	"time"
)

type OracleAPI struct {
	client *http.Client
	url    string
}

func NewOracleAPI(url string) *OracleAPI {
	return &OracleAPI{
		url: url,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (o *OracleAPI) ProposalQualities(country string) (*ProposalQualityResponse, error) {
	resp, err := o.client.Get(fmt.Sprintf("%s/api/v2/providers/quality?country=%s", o.url, country))
	if err != nil {
		log.Err(err).Msgf("failed fetching proposal qualities; country: %s", country)
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Warn().Msgf("failed fetching proposal qualities; country: %s, response code: %d", resp.StatusCode, country)
		return nil, err
	}

	var entries []ProposalQuality
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		log.Err(err).Msgf("failed fetching proposal qualities; country: %s", country)
		return nil, err
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
