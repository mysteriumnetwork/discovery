// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package quality

import (
	"time"

	"github.com/ReneKroon/ttlcache/v2"

	"github.com/mysteriumnetwork/discovery/quality/oracleapi"
)

const (
	cacheSizeLimit = 400000
)

func newCacheInstance(cacheTTL time.Duration) *ttlcache.Cache {
	cache := ttlcache.NewCache()
	cache.SetCacheSizeLimit(cacheSizeLimit)
	cache.SetTTL(cacheTTL)
	cache.SkipTTLExtensionOnHit(true)
	return cache
}

type Service struct {
	qualityAPI   *oracleapi.API
	qualityCache *ttlcache.Cache
}

func NewService(qualityAPI *oracleapi.API, cacheTTL time.Duration) *Service {
	service := &Service{
		qualityAPI:   qualityAPI,
		qualityCache: newCacheInstance(cacheTTL),
	}
	service.qualityCache.SetLoaderFunction(func(key string) (interface{}, time.Duration, error) {
		res, err := service.qualityAPI.Quality(key)
		return res, cacheTTL, err
	})
	return service
}

func (s *Service) Quality(fromCountry string) (map[string]*oracleapi.DetailedQuality, error) {
	res, err := s.qualityCache.Get(fromCountry)
	if err != nil {
		return nil, err
	}
	return res.(map[string]*oracleapi.DetailedQuality), nil
}
