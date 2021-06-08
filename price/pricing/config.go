package pricing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mysteriumnetwork/discovery/db"
	"github.com/rs/zerolog/log"
)

type ConfigProvider interface {
	Get() (Config, error)
	Update(Config) error
}

type cachedConfig struct {
	Config     Config
	ValidUntil time.Time
}

func (cc cachedConfig) isValid() bool {
	return time.Now().Before(cc.ValidUntil)
}

type ConfigProviderDB struct {
	db   *db.DB
	cc   cachedConfig
	lock sync.Mutex
}

func NewConfigProviderDB(db *db.DB) *ConfigProviderDB {
	return &ConfigProviderDB{
		db: db,
	}
}

func (cpd *ConfigProviderDB) Get() (Config, error) {
	cpd.lock.Lock()
	defer cpd.lock.Unlock()
	if cpd.cc.isValid() {
		return cpd.cc.Config, nil
	}

	cfg, err := cpd.fetchConfig()
	if err != nil {
		log.Err(err).Msg("could not fetch config")
		return Config{}, errors.New("internal error")
	}

	cpd.cc = cachedConfig{
		Config:     cfg,
		ValidUntil: time.Now().Add(time.Minute * 5),
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

	conn, err := cpd.db.Connection()
	if err != nil {
		return err
	}
	defer conn.Release()

	query := `INSERT INTO pricing_config(cfg) VALUES ($1);`
	_, err = conn.Exec(context.Background(), query, cfgJSON)
	if err != nil {
		return err
	}

	// invalidate the cache so that the pricing is updated on next Get.
	cpd.cc.ValidUntil = time.Time{}

	return nil
}

func (cpd *ConfigProviderDB) fetchConfig() (Config, error) {
	conn, err := cpd.db.Connection()
	if err != nil {
		return Config{}, err
	}
	defer conn.Release()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	res := Config{}
	row := conn.QueryRow(ctx, "SELECT cfg FROM pricing_config order by id desc limit 1;")
	err = row.Scan(&res)
	if err != nil {
		return Config{}, err
	}

	return res, nil
}

type Config struct {
	BasePrices       PriceByTypeUSD                  `json:"base_prices"`
	CountryModifiers map[ISO3166CountryCode]Modifier `json:"country_modifiers"`
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
	Residential *PriceUSD `json:"residential"`
	Other       *PriceUSD `json:"other"`
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
