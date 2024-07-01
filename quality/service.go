// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package quality

import (
	"fmt"
	"sync"
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
	qualityAPI         *oracleapi.API
	qualityCache       map[string]*oracleapi.DetailedQuality
	qualityLastUpdated time.Time
	ttl                time.Duration
	mu                 sync.Mutex
}

func NewService(qualityAPI *oracleapi.API, cacheTTL time.Duration) *Service {
	return &Service{
		qualityAPI:   qualityAPI,
		qualityCache: make(map[string]*oracleapi.DetailedQuality),
		ttl:          cacheTTL,
	}
}

func (s *Service) Quality() (map[string]*oracleapi.DetailedQuality, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if time.Since(s.qualityLastUpdated) < s.ttl {
		return s.qualityCache, nil
	}

	quality, err := s.qualityAPI.Quality()
	if err != nil {
		return nil, fmt.Errorf("failed to get quality: %w", err)
	}

	s.qualityCache = quality
	s.qualityLastUpdated = time.Now()

	return quality, nil
}
