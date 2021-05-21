package pricing

import (
	_ "embed"
	"encoding/json"
)

type ConfigProvider interface {
	Get() (Config, error)
}

//go:embed config.json
var configJSON []byte

type DefaultCountryModifiers struct {
}

func (cm *DefaultCountryModifiers) Get() (Config, error) {
	var cfg Config
	return cfg, json.Unmarshal(configJSON, &cfg)
}

type ISO3166CountryCode string
type Config struct {
	BasePrices       PriceByTypeUSD                  `json:"base_prices"`
	CountryModifiers map[ISO3166CountryCode]Modifier `json:"country_modifiers"`
}

type PriceByTypeUSD struct {
	Residential *PriceUSD `json:"residential"`
	Other       *PriceUSD `json:"other"`
}

type PriceUSD struct {
	PricePerHour float64 `json:"price_per_hour_usd"`
	PricePerGiB  float64 `json:"price_per_gib_usd"`
}

type Modifier struct {
	Residential float64
	Other       float64
}
