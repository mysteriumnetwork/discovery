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
	rdb      *redis.Client
	shutdown chan struct{}
}

func NewRepository(rdb *redis.Client) *Repository {
	return &Repository{
		rdb: rdb,
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

func (r *Repository) Start() {
	for {
		select {
		case <-time.After(20 * time.Second):
			result, err := r.rdb.ZRangeByScore(ctx, keyExpiration, &redis.ZRangeBy{
				Min: "-inf",
				Max: strconv.FormatInt(time.Now().Unix(), 10),
			}).Result()
			if err != nil {
				log.Warn().Err(err).Msg("Failed to get expired proposals")
				continue
			}
			for _, k := range result {
				r.rdb.Get(ctx, k).Val()
			}
			if err := r.rdb.Del(ctx, result...).Err(); err != nil {
				log.Warn().Err(err).Msg("Failed to delete expired proposals")
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
		Score:  float64(time.Now().Add(2 * time.Minute).Unix()),
		Member: key,
	}).Err(); err != nil {
		return err
	}
	return nil
}

func (r *Repository) Shutdown() {
	r.rdb.Shutdown(ctx)
}
