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

	"github.com/dgrijalva/jwt-go"
	"github.com/mysteriumnetwork/discovery/price/pricing"
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
	prices, err := discoveryAPI.LatestPrices()
	assert.NoError(t, err)
	assert.NotNil(t, prices.Defaults.Current.Residential.PricePerGiB)
	assert.NotNil(t, prices.Defaults.Current.Residential.PricePerHour)
	assert.NotNil(t, prices.Defaults.Current.Other.PricePerGiB)
	assert.NotNil(t, prices.Defaults.Current.Other.PricePerHour)
	assert.NotNil(t, prices.Defaults.Previous.Residential.PricePerGiB)
	assert.NotNil(t, prices.Defaults.Previous.Residential.PricePerHour)
	assert.NotNil(t, prices.Defaults.Previous.Other.PricePerGiB)
	assert.NotNil(t, prices.Defaults.Previous.Other.PricePerHour)
	assert.Greater(t, prices.CurrentValidUntil.UnixNano(), time.Unix(0, 0).UnixNano())
	assert.Greater(t, prices.PreviousValidUntil.UnixNano(), time.Unix(0, 0).UnixNano())
}

func Test_GetConfig(t *testing.T) {
	t.Run("rejected with bad token", func(t *testing.T) {
		_, err := discoveryAPI.GetPriceConfig("tkn")
		assert.Error(t, err)
		assert.Equal(t, "401", err.Error())
	})

	t.Run("fetches with correct token", func(t *testing.T) {
		cfg, err := discoveryAPI.GetPriceConfig(validToken)
		assert.NoError(t, err)

		bts, err := json.Marshal(cfg)
		assert.NoError(t, err)

		assert.JSONEq(t, expectedPricingConfig, string(bts))
	})
}

func Test_PostConfig(t *testing.T) {
	t.Run("rejected with bad token", func(t *testing.T) {
		err := discoveryAPI.UpdatePriceConfig("tkn", pricing.Config{})
		assert.Error(t, err)
		assert.Equal(t, "401", err.Error())
	})

	t.Run("rejected with invalid config", func(t *testing.T) {
		err := discoveryAPI.UpdatePriceConfig(validToken, pricing.Config{})
		assert.Error(t, err)
		assert.Equal(t, "400", err.Error())
	})

	t.Run("accepts with valid config", func(t *testing.T) {
		toSend := pricing.Config{}
		err := json.Unmarshal([]byte(expectedPricingConfig), &toSend)
		assert.NoError(t, err)
		toSend.BasePrices.Other.PricePerGiB = 11

		err = discoveryAPI.UpdatePriceConfig(validToken, toSend)
		assert.NoError(t, err)

		cfg, err := discoveryAPI.GetPriceConfig(validToken)
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
		"price_per_hour_usd": 0.00036,
		"price_per_gib_usd": 0.06
	  },
	  "other": {
		"price_per_hour_usd": 0.00036,
		"price_per_gib_usd": 0.06
	  }
	},
	"country_modifiers": {
	  "US": {
		"residential": 1.5,
		"other": 1.2
	  }
	}
}`
