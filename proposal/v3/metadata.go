// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package v3

import (
	"time"
)

// Metadata provides metadata (such as last updated timestamp) about the proposal.
// Used by the MMN.
type Metadata struct {
	ProviderID       string    `json:"provider_id"`
	ServiceType      string    `json:"service_type"`
	Country          string    `json:"country,omitempty"`
	ISP              string    `json:"isp,omitempty"`
	IPType           string    `json:"ip_type,omitempty"`
	Whitelist        bool      `json:"whitelist,omitempty"`
	MonitoringFailed bool      `json:"monitoring_failed,omitempty"`
	UpdatedAt        time.Time `json:"updated_at"`
}
