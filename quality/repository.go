package quality

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/mysteriumnetwork/discovery/proposal/v2"
)

var ctx = context.Background()

type Repository struct {
	rdb *redis.Client
}

func NewRepository(rdb *redis.Client) *Repository {
	return &Repository{
		rdb: rdb,
	}
}

func keyQuality(id, serviceType, forCountry string) string {
	return fmt.Sprintf("quality:%s:%s:%s", forCountry, serviceType, id)
}

func (r *Repository) ListQualities(providerIDS []string, serviceType, forCountry string) (map[string]v2.Quality, error) {
	keys := make([]string, len(providerIDS))
	for i, id := range providerIDS {
		keys[i] = keyQuality(id, serviceType, forCountry)
	}
	jsons, err := r.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	qualities := map[string]v2.Quality{}
	for _, j := range jsons {
		if j == nil {
			continue
		}
		s := j.(string)
		q := v2.Quality{}
		if err := json.Unmarshal([]byte(s), &q); err != nil {
			return nil, err
		}
		qualities[q.ProviderID] = q
	}
	return qualities, nil
}

func (r *Repository) StoreQuality(providerID, serviceType, forCountry string, quality float32) error {
	qualityKey := keyQuality(providerID, serviceType, forCountry)
	return r.rdb.Set(ctx, qualityKey, v2.NewQuality(providerID, quality), 0).Err()
}
