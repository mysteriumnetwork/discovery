package pricingbyservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/mysteriumnetwork/discovery/price/pricing"
	"github.com/rs/zerolog/log"
)

const PricingConfigRedisKey = "DISCOVERY_PRICE_BASE_CONFIG_BY_SERVICE"

type ConfigProvider interface {
	Get() (Config, error)
	Update(Config) error
}

type ConfigProviderDB struct {
	db   redis.UniversalClient
	lock sync.Mutex
}

func NewConfigProviderDB(redis redis.UniversalClient) *ConfigProviderDB {
	return &ConfigProviderDB{
		db: redis,
	}
}

func (cpd *ConfigProviderDB) Get() (Config, error) {
	cfg, err := cpd.fetchConfig()
	if err != nil {
		log.Err(err).Msg("could not fetch config")
		return Config{}, errors.New("internal error")
	}

	return cfg, nil
}

func (cpd *ConfigProviderDB) Update(in Config) error {
	cpd.lock.Lock()
	defer cpd.lock.Unlock()

	err := in.Validate()
	if err != nil {
		return err
	}

	cfgJSON, err := json.Marshal(in)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err = cpd.db.Set(ctx, PricingConfigRedisKey, string(cfgJSON), 0).Err()
	if err != nil {
		return err
	}

	return nil
}

func (cpd *ConfigProviderDB) fetchConfig() (Config, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	val, err := cpd.db.Get(ctx, PricingConfigRedisKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			err = cpd.db.Set(ctx, PricingConfigRedisKey, defaultPriceConfig, 0).Err()
			if err != nil {
				return Config{}, err
			}
			val = defaultPriceConfig
		} else {
			return Config{}, err
		}
	}

	res := Config{}
	return res, json.Unmarshal([]byte(val), &res)
}

type Config struct {
	BasePrices       PriceByTypeUSD                          `json:"base_prices"`
	CountryModifiers map[pricing.ISO3166CountryCode]Modifier `json:"country_modifiers"`
}

func (c Config) Validate() error {
	err := c.BasePrices.Validate()
	if err != nil {
		return fmt.Errorf("base price invalid: %w", err)
	}

	for k, v := range c.CountryModifiers {
		err := k.Validate()
		if err != nil {
			return err
		}

		err = v.Validate()
		if err != nil {
			return fmt.Errorf("country %v contains invalid pricing: %w", k, err)
		}
	}

	return nil
}

type PriceByTypeUSD struct {
	Residential *PriceByServiceTypeUSD `json:"residential"`
	Other       *PriceByServiceTypeUSD `json:"other"`
}

func (p PriceByTypeUSD) Validate() error {
	if p.Residential == nil || p.Other == nil {
		return errors.New("residential and other pricing should not be nil")
	}

	err := p.Residential.Validate()
	if err != nil {
		return err
	}

	return p.Other.Validate()
}

type PriceByServiceTypeUSD struct {
	Wireguard    *PriceUSD `json:"wireguard"`
	Scraping     *PriceUSD `json:"scraping"`
	DataTransfer *PriceUSD `json:"data_transfer"`
	DVPN         *PriceUSD `json:"dvpn"`
}

func (p PriceByServiceTypeUSD) Validate() error {
	if p.Wireguard == nil || p.Scraping == nil || p.DataTransfer == nil || p.DVPN == nil {
		return errors.New("wireguard, scraping, data_transfer and dvpn pricing should not be nil")
	}

	if err := p.Wireguard.Validate(); err != nil {
		return err
	}
	if err := p.Scraping.Validate(); err != nil {
		return err
	}
	if err := p.DVPN.Validate(); err != nil {
		return err
	}
	return p.DataTransfer.Validate()
}

type PriceUSD struct {
	PricePerHour float64 `json:"price_per_hour_usd"`
	PricePerGiB  float64 `json:"price_per_gib_usd"`
}

func (p PriceUSD) Validate() error {
	if p.PricePerGiB < 0 || p.PricePerHour < 0 {
		return errors.New("prices should be non negative")
	}

	return nil
}

type Modifier struct {
	Residential float64 `json:"residential"`
	Other       float64 `json:"other"`
}

func (m Modifier) Validate() error {
	if m.Residential < 0 || m.Other < 0 {
		return errors.New("modifiers should be non negative")
	}
	return nil
}

var defaultPriceConfig = `{
	"base_prices": {
	  "residential": {
		"wireguard": {
			"price_per_hour_usd": 0.00036,
			"price_per_gib_usd": 0.06
		},
		"scraping": {
			"price_per_hour_usd": 0.00036,
			"price_per_gib_usd": 0.06
		},
		"data_transfer": {
			"price_per_hour_usd": 0.00036,
			"price_per_gib_usd": 0.06
		},
		"dvpn": {
			"price_per_hour_usd": 0.00036,
			"price_per_gib_usd": 0.06
		}
	  },
	  "other": {
		"wireguard": {
			"price_per_hour_usd": 0.00036,
			"price_per_gib_usd": 0.06
		},
		"scraping": {
			"price_per_hour_usd": 0.00036,
			"price_per_gib_usd": 0.06
		},
		"data_transfer": {
			"price_per_hour_usd": 0.00036,
			"price_per_gib_usd": 0.06
		},
		"dvpn": {
			"price_per_hour_usd": 0.00036,
			"price_per_gib_usd": 0.06
		}
	  }
	},
	"country_modifiers": {
	  "US": {
		"residential": 1.5,
		"other": 1.2
	  }
	}
  }`
