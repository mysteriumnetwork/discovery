// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package e2e

import (
	_ "embed"
	"math/big"
	"testing"

	"github.com/dghubble/sling"
	v2 "github.com/mysteriumnetwork/discovery/proposal/v2"
	"github.com/stretchr/testify/assert"
)

func Test_ProposalFiltering(t *testing.T) {
	api := newAPI("http://localhost:8080/")

	t.Run("provider_id", func(t *testing.T) {
		query := query{
			ProviderID:  "0xd0cd77e69b572a638ca2d6d881e09bb7d5558c69",
			ServiceType: "wireguard",
		}
		proposals, err := api.ListFilters(query)
		assert.NoError(t, err)
		assert.Len(t, proposals, 1)
		for _, p := range proposals {
			assert.Equal(t, query.ProviderID, p.ProviderID)
			assert.Equal(t, query.ServiceType, p.ServiceType)
		}
	})

	t.Run("country", func(t *testing.T) {
		for _, query := range []query{
			{Country: "LT"},
			{Country: "RU"},
			{Country: "US"},
		} {
			proposals, err := api.ListFilters(query)
			assert.NoError(t, err)
			assert.True(t, len(proposals) > 0)
			for _, p := range proposals {
				assert.Equal(t, query.Country, p.Location.Country)
			}
		}
	})

	t.Run("compatibility", func(t *testing.T) {
		for _, query := range []query{
			{CompatibilityMin: 1, CompatibilityMax: 1},
			{CompatibilityMin: 0, CompatibilityMax: 2},
		} {
			proposals, err := api.ListFilters(query)
			assert.NoError(t, err)
			assert.True(t, len(proposals) > 0)
			for _, p := range proposals {
				assert.True(t, p.Compatibility >= query.CompatibilityMin && p.Compatibility <= query.CompatibilityMax, p.Compatibility)
			}
		}
	})

	t.Run("service_type", func(t *testing.T) {
		for _, query := range []query{
			{ServiceType: "wireguard"},
			{ServiceType: "openvpn"},
		} {
			proposals, err := api.ListFilters(query)
			assert.NoError(t, err)
			assert.True(t, len(proposals) > 0)
			for _, p := range proposals {
				assert.Equal(t, query.ServiceType, p.ServiceType)
			}
		}
	})

	t.Run("ip_type", func(t *testing.T) {
		proposals, err := api.ListFilters(query{IPType: "residential"})
		assert.NoError(t, err)
		assert.True(t, len(proposals) > 0)
		for _, p := range proposals {
			assert.True(t, p.Location.IPType.IsResidential())
		}
	})

	t.Run("price_hour_max", func(t *testing.T) {
		for _, query := range []query{
			{PriceHourMax: 300000000000000},
			{PriceHourMax: 850000000000000},
		} {
			proposals, err := api.ListFilters(query)
			assert.NoError(t, err)
			assert.True(t, len(proposals) > 0, "no results matching query: %+v", query)
			for _, proposal := range proposals {
				assert.NotNil(t, proposal.Price.PerHour)

				cmp := proposal.Price.PerHour.Cmp(big.NewInt(query.PriceHourMax))
				assert.True(t, cmp == -1 || cmp == 0)
			}
		}

		proposals, err := api.ListFilters(query{PriceHourMax: 110})
		assert.NoError(t, err)
		assert.Len(t, proposals, 0)
	})

	t.Run("price_gib_max", func(t *testing.T) {
		for _, query := range []query{
			{PriceGibMax: 220000029504303120},
			{PriceGibMax: 310000029504303120},
		} {
			proposals, err := api.ListFilters(query)
			assert.NoError(t, err)
			assert.True(t, len(proposals) > 0)
			for _, proposal := range proposals {
				assert.NotNil(t, proposal.Price.PerGiB)

				cmp := proposal.Price.PerGiB.Cmp(big.NewInt(query.PriceGibMax))
				assert.True(t, cmp == -1 || cmp == 0)
			}
		}

		proposals, err := api.ListFilters(query{PriceGibMax: 110})
		assert.NoError(t, err)
		assert.Len(t, proposals, 0)
	})

	t.Run("quality", func(t *testing.T) {
		for _, query := range []query{
			{QualityMin: 1.0},
			{QualityMin: 1.4},
			{QualityMin: 2.5},
		} {
			proposals, err := api.ListFilters(query)
			assert.NoError(t, err)
			assert.True(t, len(proposals) > 0)
			for _, proposal := range proposals {
				assert.GreaterOrEqual(t, proposal.Quality, query.QualityMin)
			}
		}
	})
}

func newAPI(basePath string) *api {
	return &api{
		basePath: basePath,
	}
}

type api struct {
	basePath string
}

func (a *api) ListFilters(query query) (proposals []v2.Proposal, err error) {
	_, err = sling.New().Base(a.basePath).Get("/api/v3/proposals").QueryStruct(query).Receive(&proposals, nil)
	return proposals, err
}

type query struct {
	From               string  `url:"from"`
	ProviderID         string  `url:"provider_id"`
	ServiceType        string  `url:"service_type"`
	Country            string  `url:"country"`
	IPType             string  `url:"ip_type"`
	AccessPolicy       string  `url:"access_policy"`
	AccessPolicySource string  `url:"access_policy_source"`
	PriceGibMax        int64   `url:"price_gib_max"`
	PriceHourMax       int64   `url:"price_hour_max"`
	CompatibilityMin   int     `url:"compatibility_min"`
	CompatibilityMax   int     `url:"compatibility_max"`
	QualityMin         float32 `url:"quality_min"`
}
