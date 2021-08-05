package tags

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type API struct {
	url    string
	client *http.Client
}

type TagResponse struct {
	Tags []string `json:"tags,omitempty"`
}

func NewApi(url string) *API {
	return &API{
		url: url,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (api *API) GetTags(providerID string) ([]string, error) {
	resp, err := api.client.Get(fmt.Sprintf("%v/api/v1/tags/provider/%v", api.url, providerID))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("received response code: %v", resp.StatusCode)
	}

	var res TagResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return res.Tags, nil
}
