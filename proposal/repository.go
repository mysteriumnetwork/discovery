// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package proposal

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	v2 "github.com/mysteriumnetwork/discovery/proposal/v2"
	"github.com/rs/zerolog/log"
)

var ctx = context.Background()

type Repository struct {
	rdb                *redis.Client
	shutdown           chan struct{}
	expirationJobDelay time.Duration
	expirationDuration time.Duration
}

func NewRepository(rdb *redis.Client) *Repository {
	return &Repository{
		rdb:                rdb,
		expirationDuration: 2 * time.Minute,
		expirationJobDelay: 20 * time.Second,
	}
}

func keyProposals(serviceType, id string) string {
	return fmt.Sprintf("proposals:%s:%s", serviceType, id)
}

func keyServiceType(serviceType string) string {
	return "service-type:" + serviceType
}

func keyCountry(country string) string {
	return "country:" + country
}

const keyExpiration = "expiration"

func (r *Repository) StartExpirationJob() {
	for {
		select {
		case <-time.After(r.expirationJobDelay):
			result, err := r.rdb.ZRangeByScore(ctx, keyExpiration, &redis.ZRangeBy{
				Min: "-inf",
				Max: strconv.FormatInt(time.Now().Unix(), 10),
			}).Result()
			if err != nil {
				log.Warn().Err(err).Msg("Failed to get expired proposals")
				continue
			}
			if err := r.Delete(result...); err != nil {
				log.Err(err).Msgf("Failed to delete proposals %v", result)
			}

		case <-r.shutdown:
			return
		}
	}
}

func (r *Repository) List(serviceType, country string) ([]v2.Proposal, error) {
	keys, err := r.rdb.SInter(ctx, keyServiceType(serviceType), keyCountry(country)).Result()
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, nil
	}

	jsonProposals, err := r.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	var proposals []v2.Proposal
	for _, j := range jsonProposals {
		s := j.(string)
		p := v2.Proposal{}
		if err := json.Unmarshal([]byte(s), &p); err != nil {
			return nil, err
		}
		proposals = append(proposals, p)
	}

	return proposals, nil
}

func (r *Repository) Store(id string, serviceType string, country string, proposal v2.Proposal) error {
	key := keyProposals(serviceType, id)
	err := r.rdb.Set(ctx, key, proposal, 0).Err()
	if err != nil {
		return err
	}
	if err := r.rdb.SAdd(ctx, keyCountry(country), key).Err(); err != nil {
		return err
	}
	if err := r.rdb.SAdd(ctx, keyServiceType(serviceType), key).Err(); err != nil {
		return err
	}
	if err := r.rdb.ZAdd(ctx, keyExpiration, &redis.Z{
		Score:  float64(time.Now().Add(r.expirationDuration).Unix()),
		Member: key,
	}).Err(); err != nil {
		return err
	}
	return nil
}

func (r *Repository) Delete(keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	for _, k := range keys {
		j := r.rdb.Get(ctx, k).Val()
		var p v2.Proposal
		if err := json.Unmarshal([]byte(j), &p); err != nil {
			log.Warn().Err(err).Msgf("Failed to unmarshal %s %v", k, j)
			continue
		}
		if err := r.rdb.SRem(ctx, keyCountry(p.Location.Country), k).Err(); err != nil {
			log.Warn().Err(err).Msgf("Failed to delete %s from country index", k)
		}
		if err := r.rdb.SRem(ctx, keyServiceType(p.ServiceType), k).Err(); err != nil {
			log.Warn().Err(err).Msgf("Failed to delete %s from serviceType index", k)
		}
	}
	if err := r.rdb.ZRem(ctx, keyExpiration, keys).Err(); err != nil {
		return err
	}
	if err := r.rdb.Del(ctx, keys...).Err(); err != nil {
		log.Warn().Err(err).Msgf("Failed to delete proposal keys %v", keys)
	}
	return nil
}

func (r *Repository) Shutdown() {
	r.rdb.Shutdown(ctx)
}
