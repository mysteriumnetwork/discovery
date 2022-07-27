package pricing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

type PriceGetter struct {
	db    redis.UniversalClient
	lp    LatestPrices
	mutex sync.Mutex
}

func NewPriceGetter(db redis.UniversalClient) (*PriceGetter, error) {
	loaded, err := loadPricing(db)
	if err != nil {
		return nil, fmt.Errorf("could not laod initial price %w", err)
	}

	return &PriceGetter{
		db: db,
		lp: loaded,
	}, nil
}

func loadPricing(db redis.UniversalClient) (LatestPrices, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	redisRes, err := db.Get(ctx, PriceRedisKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			log.Info().Msg("no pricing found in redis, will use defaults")
			cp := defaultPrices
			now := time.Now().UTC()
			cp.CurrentValidUntil = now.Add(time.Second * 1)
			cp.PreviousValidUntil = now.Add(time.Second * -1)
			return cp, nil
		}
		return LatestPrices{}, err
	}

	var res LatestPrices
	return res, json.Unmarshal([]byte(redisRes), &res)
}

func (pg *PriceGetter) GetPrices() LatestPrices {
	pg.mutex.Lock()
	defer pg.mutex.Unlock()

	if time.Now().UTC().After(pg.lp.CurrentValidUntil) {
		loaded, err := loadPricing(pg.db)
		if err != nil {
			log.Err(err).Msg("could not load pricing from db")
			return pg.lp.WithCurrentTime()
		}
		log.Info().Msg("pricing loaded from db")
		pg.lp = loaded
	}

	return pg.lp.WithCurrentTime()
}

var defaultPrices = LatestPrices{
	Defaults: &PriceHistory{
		Current: &PriceByType{
			Residential: &Price{
				PricePerHour:              big.NewInt(900000000000000),
				PricePerHourHumanReadable: 0.0009,
				PricePerGiB:               big.NewInt(150000000000000000),
				PricePerGiBHumanReadable:  0.15,
			},
			Other: &Price{
				PricePerHour:              big.NewInt(900000000000000),
				PricePerHourHumanReadable: 0.0009,
				PricePerGiB:               big.NewInt(150000000000000000),
				PricePerGiBHumanReadable:  0.15,
			},
		},
		Previous: &PriceByType{
			Residential: &Price{
				PricePerHour:              big.NewInt(900000000000000),
				PricePerHourHumanReadable: 0.0009,
				PricePerGiB:               big.NewInt(150000000000000000),
				PricePerGiBHumanReadable:  0.15,
			},
			Other: &Price{
				PricePerHour:              big.NewInt(900000000000000),
				PricePerHourHumanReadable: 0.0009,
				PricePerGiB:               big.NewInt(150000000000000000),
				PricePerGiBHumanReadable:  0.15,
			},
		},
	},
}
