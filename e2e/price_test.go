// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package e2e

import (
	_ "embed"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
	assert.Greater(t, len(prices.PerCountry), 0)
}
