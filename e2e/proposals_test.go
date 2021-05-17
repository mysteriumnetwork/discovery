// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package e2e

import (
	_ "embed"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_ProposalFiltering(t *testing.T) {
	err := purgeProposalsDB()
	assert.NoError(t, err)
	templates, err := publishProposals(t)
	assert.NoError(t, err)

	api := discoveryAPI

	t.Run("provider_id", func(t *testing.T) {
		query := Query{
			ProviderID:  "0x1",
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
		for _, query := range []Query{
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
		for _, query := range []Query{
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
		for _, query := range []Query{
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
		proposals, err := api.ListFilters(Query{IPType: "residential"})
		assert.NoError(t, err)
		assert.True(t, len(proposals) > 0)
		for _, p := range proposals {
			assert.True(t, p.Location.IPType.IsResidential())
		}
	})

	t.Run("price_hour_max", func(t *testing.T) {
		for _, query := range []Query{
			{PriceHourMax: 5},
			{PriceHourMax: 15},
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

		proposals, err := api.ListFilters(Query{PriceHourMax: 4})
		assert.NoError(t, err)
		assert.Len(t, proposals, 0)
	})

	t.Run("price_gib_max", func(t *testing.T) {
		for _, query := range []Query{
			{PriceGibMax: 15},
			{PriceGibMax: 20},
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

		proposals, err := api.ListFilters(Query{PriceGibMax: 5})
		assert.NoError(t, err)
		assert.Len(t, proposals, 0)
	})

	t.Run("quality", func(t *testing.T) {
		for _, query := range []Query{
			{QualityMin: 1.0},
			{QualityMin: 1.4},
			{QualityMin: 2.5},
		} {
			proposals, err := api.ListFilters(query)
			assert.NoError(t, err)
			assert.True(t, len(proposals) > 0)
			for _, proposal := range proposals {
				assert.GreaterOrEqual(t, proposal.Quality.Quality, query.QualityMin)
			}
		}
	})

	t.Run("unregister", func(t *testing.T) {
		for _, id := range []string{"0x1", "0x2", "0x3"} {
			proposals, err := api.ListFilters(Query{ProviderID: id})
			assert.NoError(t, err)
			assert.Len(t, proposals, 1)
			assert.True(t, proposals[0].ProviderID == id, "missing ProviderID %s in response", id)
		}

		for _, pt := range templates {
			assert.NoError(t, pt.unregister())
		}

		assert.Eventuallyf(t, func() bool {
			proposals, err := discoveryAPI.ListFilters(Query{})
			assert.NoError(t, err)
			return len(proposals) == 0
		}, time.Second*10, time.Millisecond*500, "proposals did not unregister")
	})
}

func publishProposals(t *testing.T) ([]*template, error) {
	templates := []*template{
		newTemplate().providerID("0x1").country("LT").compatibility(0).serviceType("wireguard").prices(10, 5),
		newTemplate().providerID("0x2").country("RU").compatibility(1).prices(20, 10),
		newTemplate().providerID("0x3").country("US").compatibility(2).prices(30, 15),
	}
	for _, t := range templates {
		if err := t.publishPing(); err != nil {
			return nil, err
		}
	}
	assert.Eventuallyf(t, func() bool {
		proposals, err := discoveryAPI.ListFilters(Query{})
		assert.NoError(t, err)
		return len(proposals) == len(templates)
	}, time.Second*10, time.Millisecond*500, "publishing did not seed")
	return templates, nil
}
