// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package e2e

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/mysteriumnetwork/discovery/price/pricingbyservice"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		fmt.Println("skipping e2e, 'short' flag set")
		return
	}

	os.Exit(m.Run())
}

func Test_LatestPrices(t *testing.T) {
	prices, err := PricerAPI.LatestPrices()
	assert.NoError(t, err)
	assert.NotNil(t, prices.Defaults.Current.Residential.DVPN.PricePerGiB)
	assert.NotNil(t, prices.Defaults.Current.Residential.DVPN.PricePerHour)
	assert.NotNil(t, prices.Defaults.Current.Residential.Scraping.PricePerGiB)
	assert.NotNil(t, prices.Defaults.Current.Residential.Scraping.PricePerHour)
	assert.NotNil(t, prices.Defaults.Current.Residential.Wireguard.PricePerGiB)
	assert.NotNil(t, prices.Defaults.Current.Residential.Wireguard.PricePerHour)
	assert.NotNil(t, prices.Defaults.Current.Residential.DataTransfer.PricePerGiB)
	assert.NotNil(t, prices.Defaults.Current.Residential.DataTransfer.PricePerHour)
	assert.NotNil(t, prices.Defaults.Current.Other.DVPN.PricePerGiB)
	assert.NotNil(t, prices.Defaults.Current.Other.DVPN.PricePerHour)
	assert.NotNil(t, prices.Defaults.Current.Other.Scraping.PricePerGiB)
	assert.NotNil(t, prices.Defaults.Current.Other.Scraping.PricePerHour)
	assert.NotNil(t, prices.Defaults.Current.Other.Wireguard.PricePerGiB)
	assert.NotNil(t, prices.Defaults.Current.Other.Wireguard.PricePerHour)
	assert.NotNil(t, prices.Defaults.Current.Other.DataTransfer.PricePerGiB)
	assert.NotNil(t, prices.Defaults.Current.Other.DataTransfer.PricePerHour)
	assert.NotNil(t, prices.Defaults.Previous.Residential.DVPN.PricePerGiB)
	assert.NotNil(t, prices.Defaults.Previous.Residential.DVPN.PricePerHour)
	assert.NotNil(t, prices.Defaults.Previous.Residential.Scraping.PricePerGiB)
	assert.NotNil(t, prices.Defaults.Previous.Residential.Scraping.PricePerHour)
	assert.NotNil(t, prices.Defaults.Previous.Residential.Wireguard.PricePerGiB)
	assert.NotNil(t, prices.Defaults.Previous.Residential.Wireguard.PricePerHour)
	assert.NotNil(t, prices.Defaults.Previous.Residential.DataTransfer.PricePerGiB)
	assert.NotNil(t, prices.Defaults.Previous.Residential.DataTransfer.PricePerHour)
	assert.NotNil(t, prices.Defaults.Previous.Other.DVPN.PricePerGiB)
	assert.NotNil(t, prices.Defaults.Previous.Other.DVPN.PricePerHour)
	assert.NotNil(t, prices.Defaults.Previous.Other.Scraping.PricePerGiB)
	assert.NotNil(t, prices.Defaults.Previous.Other.Scraping.PricePerHour)
	assert.NotNil(t, prices.Defaults.Previous.Other.Wireguard.PricePerGiB)
	assert.NotNil(t, prices.Defaults.Previous.Other.Wireguard.PricePerHour)
	assert.NotNil(t, prices.Defaults.Previous.Other.DataTransfer.PricePerGiB)
	assert.NotNil(t, prices.Defaults.Previous.Other.DataTransfer.PricePerHour)

	assert.Greater(t, prices.CurrentValidUntil.UnixNano(), time.Unix(0, 0).UnixNano())
	assert.Greater(t, prices.PreviousValidUntil.UnixNano(), time.Unix(0, 0).UnixNano())
	assert.Greater(t, prices.CurrentServerTime.UnixNano(), time.Unix(0, 0).UnixNano())
}

func Test_GetConfig(t *testing.T) {
	t.Run("rejected with bad token", func(t *testing.T) {
		_, err := PricerAPI.GetPriceConfig("tkn")
		assert.Error(t, err)
		assert.Equal(t, "401", err.Error())
	})

	t.Run("fetches with correct token", func(t *testing.T) {
		cfg, err := PricerAPI.GetPriceConfig(validToken)
		assert.NoError(t, err)

		bts, err := json.Marshal(cfg)
		assert.NoError(t, err)

		assert.JSONEq(t, expectedPricingConfig, string(bts))
	})
}

func Test_PostConfig(t *testing.T) {
	t.Run("rejected with bad token", func(t *testing.T) {
		err := PricerAPI.UpdatePriceConfig("tkn", pricingbyservice.Config{})
		assert.Error(t, err)
		assert.Equal(t, "401", err.Error())
	})

	t.Run("rejected with invalid config", func(t *testing.T) {
		err := PricerAPI.UpdatePriceConfig(validToken, pricingbyservice.Config{})
		assert.Error(t, err)
		assert.Equal(t, "400", err.Error())
	})

	t.Run("accepts with valid config", func(t *testing.T) {
		toSend := pricingbyservice.Config{}
		err := json.Unmarshal([]byte(expectedPricingConfig), &toSend)
		assert.NoError(t, err)
		toSend.BasePrices.Other.Wireguard.PricePerGiB = 11

		err = PricerAPI.UpdatePriceConfig(validToken, toSend)
		assert.NoError(t, err)

		cfg, err := PricerAPI.GetPriceConfig(validToken)
		assert.NoError(t, err)
		assert.EqualValues(t, toSend, cfg)
	})
}

func issueBearerToken() (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(time.Hour * 24 * 30).Unix(),
	})
	s, err := token.SignedString([]byte("suchsecret"))
	return s, err
}

var validToken string

func init() {
	tkn, _ := issueBearerToken()
	validToken = tkn
}

var expectedPricingConfig = `
{
    "base_prices": {
        "residential": {
            "wireguard": {
                "price_per_hour_usd": 0.00005,
                "price_per_gib_usd": 0.2
            },
            "scraping": {
                "price_per_hour_usd": 0.00005,
                "price_per_gib_usd": 0.191
            },
            "data_transfer": {
                "price_per_hour_usd": 0.00005,
                "price_per_gib_usd": 0.016
            },
            "dvpn": {
                "price_per_hour_usd": 0.00005,
                "price_per_gib_usd": 0.05
            }
        },
        "other": {
            "wireguard": {
                "price_per_hour_usd": 0.00005,
                "price_per_gib_usd": 0.09
            },
            "scraping": {
                "price_per_hour_usd": 0.00005,
                "price_per_gib_usd": 0.0101
            },
            "data_transfer": {
                "price_per_hour_usd": 0.00005,
                "price_per_gib_usd": 0.012
            },
            "dvpn": {
                "price_per_hour_usd": 0.00005,
                "price_per_gib_usd": 0.03
            }
        }
    },
    "country_modifiers": {
        "US": {
            "residential": 1,
            "other": 1
        }
    }
}`
