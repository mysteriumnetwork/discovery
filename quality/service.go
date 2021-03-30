// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package quality

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/mysteriumnetwork/discovery/quality/oracleapi"
)

const cacheDuration = 20 * time.Second

var ctx = context.Background()

type Service struct {
	qualityAPI *oracleapi.API
	rdb        *redis.Client
}

func NewService(qualityAPI *oracleapi.API, rdb *redis.Client) *Service {
	return &Service{
		qualityAPI: qualityAPI,
		rdb:        rdb,
	}
}

func keyQuality(fromCountry string) string {
	return fmt.Sprintf("quality:%s", fromCountry)
}

func (s *Service) Quality(fromCountry string) (*oracleapi.ProposalQualityResponse, error) {
	res := s.rdb.Get(ctx, keyQuality(fromCountry))
	if res.Err() != nil {
		quality, err := s.qualityAPI.Quality(fromCountry)
		if err != nil {
			return nil, err
		}
		s.rdb.SetNX(ctx, keyQuality(fromCountry), quality, cacheDuration)
		return quality, nil
	}
	result := oracleapi.ProposalQualityResponse{}
	if err := res.Scan(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
