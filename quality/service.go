// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package quality

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/allegro/bigcache"
	"github.com/mysteriumnetwork/discovery/quality/oracleapi"
	"github.com/rs/zerolog/log"
)

const cacheDuration = 20 * time.Second

var cache *bigcache.BigCache

func init() {
	cfg := bigcache.DefaultConfig(cacheDuration)
	cfg.CleanWindow = 10 * time.Second
	cfg.Verbose = true
	cache, _ = bigcache.NewBigCache(cfg)
}

type Service struct {
	qualityAPI *oracleapi.API
}

func NewService(qualityAPI *oracleapi.API) *Service {
	return &Service{
		qualityAPI: qualityAPI,
	}
}

func keyQuality(fromCountry string) string {
	return fmt.Sprintf("quality:%s", fromCountry)
}

func (s *Service) Quality(fromCountry string) (*oracleapi.ProposalQualityResponse, error) {
	res, err := cache.Get(keyQuality(fromCountry))
	if err != nil {
		quality, err := s.qualityAPI.Quality(fromCountry)
		if err != nil {
			return nil, err
		}
		response, err := json.Marshal(quality)
		if err != nil {
			log.Err(err).Msg("Failed to marshal quality response for caching")
		} else if err := cache.Set(keyQuality(fromCountry), response); err != nil {
			log.Err(err).Msg("Failed to cache quality response")
		}
		return quality, nil
	}
	result := oracleapi.ProposalQualityResponse{}
	if err := json.Unmarshal(res, &result); err != nil {
		return nil, fmt.Errorf("failed to decode from cache: %w", err)
	}
	return &result, nil
}
