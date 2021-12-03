// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package quality

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/mysteriumnetwork/discovery/quality/oracleapi"
	"github.com/rs/zerolog/log"
)

const cacheDuration = 20 * time.Second

var qualityCache *bigcache.BigCache
var sessionCache *bigcache.BigCache
var latencyCache *bigcache.BigCache
var bandwidthCache *bigcache.BigCache

func init() {
	cfg := bigcache.DefaultConfig(cacheDuration)
	cfg.CleanWindow = 10 * time.Second
	cfg.HardMaxCacheSize = 128
	cfg.Verbose = true
	qualityCache, _ = bigcache.NewBigCache(cfg)
	sessionCache, _ = bigcache.NewBigCache(cfg)
	latencyCache, _ = bigcache.NewBigCache(cfg)
	bandwidthCache, _ = bigcache.NewBigCache(cfg)
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

func keyLatency(fromCountry string) string {
	return fmt.Sprintf("latency:%s", fromCountry)
}

func keyBandwidth(fromCountry string) string {
	return fmt.Sprintf("bandwidth:%s", fromCountry)
}

func keySessions(fromCountry string) string {
	return fmt.Sprintf("session:%s", fromCountry)
}

func (s *Service) Quality(fromCountry string) (*oracleapi.ProposalQualityResponse, error) {
	res, err := qualityCache.Get(keyQuality(fromCountry))
	if err != nil {
		quality, err := s.qualityAPI.Quality(fromCountry)
		if err != nil {
			return nil, err
		}
		response, err := json.Marshal(quality)
		if err != nil {
			log.Err(err).Msg("Failed to marshal quality response for caching")
		} else if err := qualityCache.Set(keyQuality(fromCountry), response); err != nil {
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

func (s *Service) Sessions(fromCountry string) (*oracleapi.SessionsResponse, error) {
	res, err := sessionCache.Get(keySessions(fromCountry))
	if err != nil {
		sessions, err := s.qualityAPI.Sessions(fromCountry)
		if err != nil {
			return nil, err
		}
		response, err := json.Marshal(sessions)
		if err != nil {
			log.Err(err).Msg("Failed to marshal sessions response for caching")
		} else if err := sessionCache.Set(keySessions(fromCountry), response); err != nil {
			log.Err(err).Msg("Failed to cache sessions response")
		}
		return sessions, nil
	}
	result := oracleapi.SessionsResponse{}
	if err := json.Unmarshal(res, &result); err != nil {
		return nil, fmt.Errorf("failed to decode from cache: %w", err)
	}
	return &result, nil
}

func (s *Service) Latency(fromCountry string) (*oracleapi.LatencyResponse, error) {
	res, err := latencyCache.Get(keyLatency(fromCountry))
	if err != nil {
		response, err := s.qualityAPI.Latency(fromCountry)
		if err != nil {
			return nil, err
		}
		marshaled, err := json.Marshal(response)
		if err != nil {
			log.Err(err).Msg("Failed to marshal latency response for caching")
		} else if err := latencyCache.Set(keyLatency(fromCountry), marshaled); err != nil {
			log.Err(err).Msg("Failed to cache latency response")
		}
		return response, nil
	}
	cached := oracleapi.LatencyResponse{}
	if err := json.Unmarshal(res, &cached); err != nil {
		return nil, fmt.Errorf("failed to decode from cache: %w", err)
	}
	return &cached, nil
}

func (s *Service) Bandwidth(fromCountry string) (*oracleapi.BandwidthResponse, error) {
	res, err := bandwidthCache.Get(keyBandwidth(fromCountry))
	if err != nil {
		response, err := s.qualityAPI.Bandwidth(fromCountry)
		if err != nil {
			return nil, err
		}
		marshaled, err := json.Marshal(response)
		if err != nil {
			log.Err(err).Msg("Failed to marshal bandwidth response for caching")
		} else if err := bandwidthCache.Set(keyBandwidth(fromCountry), marshaled); err != nil {
			log.Err(err).Msg("Failed to cache bandwidth response")
		}
		return response, nil
	}
	cached := oracleapi.BandwidthResponse{}
	if err := json.Unmarshal(res, &cached); err != nil {
		return nil, fmt.Errorf("failed to decode from cache: %w", err)
	}
	return &cached, nil
}
