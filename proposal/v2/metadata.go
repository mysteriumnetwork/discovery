// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package v2

import (
	"math/big"
	"time"
)

// Metadata provides metadata (such as last updated timestamp) about the proposal.
// Used by the MMN.
type Metadata struct {
	ProviderID   string    `json:"provider_id"`
	ServiceType  string    `json:"service_type"`
	Country      *string   `json:"country"`
	ISP          *string   `json:"isp"`
	IPType       *string   `json:"ip_type"`
	Whitelist    bool      `json:"whitelist"`
	PricePerGib  *big.Int  `json:"price_per_gib" swaggertype:"integer"`
	PricePerHour *big.Int  `json:"price_per_hour" swaggertype:"integer"`
	UpdatedAt    time.Time `json:"updated_at"`
}
