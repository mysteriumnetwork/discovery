// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package quality

import (
	"time"

	"github.com/ReneKroon/ttlcache/v2"

	"github.com/mysteriumnetwork/discovery/quality/oracleapi"
)

const cacheTTL = 20 * time.Second
const cacheSizeLimit = 400000

func newCacheInstance() *ttlcache.Cache {
	cache := ttlcache.NewCache()
	cache.SetCacheSizeLimit(cacheSizeLimit)
	cache.SetTTL(cacheTTL)
	cache.SkipTTLExtensionOnHit(true)
	return cache
}

type Service struct {
	qualityAPI     *oracleapi.API
	qualityCache   *ttlcache.Cache
	sessionCache   *ttlcache.Cache
	latencyCache   *ttlcache.Cache
	bandwidthCache *ttlcache.Cache
}

func NewService(qualityAPI *oracleapi.API) *Service {
	service := &Service{
		qualityAPI:     qualityAPI,
		qualityCache:   newCacheInstance(),
		sessionCache:   newCacheInstance(),
		latencyCache:   newCacheInstance(),
		bandwidthCache: newCacheInstance(),
	}
	service.qualityCache.SetLoaderFunction(func(key string) (interface{}, time.Duration, error) {
		res, err := service.qualityAPI.Quality(key)
		return res, cacheTTL, err
	})
	service.sessionCache.SetLoaderFunction(func(key string) (interface{}, time.Duration, error) {
		res, err := service.qualityAPI.Sessions(key)
		return res, cacheTTL, err
	})
	service.latencyCache.SetLoaderFunction(func(key string) (interface{}, time.Duration, error) {
		res, err := service.qualityAPI.Latency(key)
		return res, cacheTTL, err
	})
	service.bandwidthCache.SetLoaderFunction(func(key string) (interface{}, time.Duration, error) {
		res, err := service.qualityAPI.Bandwidth(key)
		return res, cacheTTL, err
	})
	return service
}

func (s *Service) Quality(fromCountry string) (*oracleapi.ProposalQualityResponse, error) {
	res, err := s.qualityCache.Get(fromCountry)
	if err != nil {
		return nil, err
	}
	return res.(*oracleapi.ProposalQualityResponse), nil
}

func (s *Service) Sessions(fromCountry string) (*oracleapi.SessionsResponse, error) {
	res, err := s.sessionCache.Get(fromCountry)
	if err != nil {
		return nil, err
	}
	return res.(*oracleapi.SessionsResponse), nil
}

func (s *Service) Latency(fromCountry string) (*oracleapi.LatencyResponse, error) {
	res, err := s.latencyCache.Get(fromCountry)
	if err != nil {
		return nil, err
	}
	return res.(*oracleapi.LatencyResponse), nil
}

func (s *Service) Bandwidth(fromCountry string) (*oracleapi.BandwidthResponse, error) {
	res, err := s.bandwidthCache.Get(fromCountry)
	if err != nil {
		return nil, err
	}
	return res.(*oracleapi.BandwidthResponse), nil
}
