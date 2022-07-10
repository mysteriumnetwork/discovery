// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package v3

import (
	"encoding/json"
	"math/big"
	"reflect"
)

type ProposalPingMessage struct {
	Proposal Proposal `json:"proposal"`
}

func (p ProposalPingMessage) IsEmpty() bool {
	return reflect.DeepEqual(p, ProposalPingMessage{})
}

type ProposalUnregisterMessage struct {
	Proposal Proposal `json:"proposal"`
}

func (p ProposalUnregisterMessage) IsEmpty() bool {
	return reflect.DeepEqual(p, ProposalUnregisterMessage{})
}

func (p ProposalUnregisterMessage) Key() string {
	return p.Proposal.ServiceType + ":" + p.Proposal.ProviderID
}

const Format = "service-proposal/v2"

type Proposal struct {
	ID             int            `json:"id"`
	Format         string         `json:"format"`
	Compatibility  int            `json:"compatibility"`
	ProviderID     string         `json:"provider_id"`
	ServiceType    string         `json:"service_type"`
	Location       Location       `json:"location"`
	Contacts       []Contact      `json:"contacts"`
	AccessPolicies []AccessPolicy `json:"access_policies,omitempty"`
	Quality        Quality        `json:"quality"`
	Tags           []string       `json:"tags,omitempty"`
}

func NewProposal(providerID, serviceType string) *Proposal {
	return &Proposal{
		Format:      Format,
		ProviderID:  providerID,
		ServiceType: serviceType,
	}
}

func (p Proposal) Key() string {
	return p.ProviderID + "." + p.ServiceType
}

func (p Proposal) MarshalBinary() (data []byte, err error) {
	marshal, err := json.Marshal(p)
	return marshal, err
}

func (p Proposal) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &p)
}

type IPType string

func (ipt IPType) IsResidential() bool {
	switch ipt {
	case "residential", "cellular":
		return true
	default:
		return false
	}
}

type Location struct {
	Continent string `json:"continent,omitempty"`
	Country   string `json:"country,omitempty"`
	City      string `json:"city,omitempty"`
	ASN       int    `json:"asn,omitempty"`
	ISP       string `json:"isp,omitempty"`
	IPType    IPType `json:"ip_type,omitempty"`
}

type Price struct {
	PerHour *big.Int `json:"per_hour" swaggertype:"integer"`
	PerGiB  *big.Int `json:"per_gib" swaggertype:"integer"`
}

type Contact struct {
	Type       string           `json:"type"`
	Definition *json.RawMessage `json:"definition" swaggertype:"object"`
}

// AccessPolicy represents the access controls for proposal
type AccessPolicy struct {
	ID     string `json:"id"`
	Source string `json:"source"`
}

type Quality struct {
	// Quality valuation from the oracle.
	Quality float64 `json:"quality"`
	// Latency in ms.
	Latency float64 `json:"latency"`
	// Bandwidth in Mbps.
	Bandwidth float64 `json:"bandwidth"`
	// Uptime
	Uptime float64 `json:"uptime"`
	// MonitoringFailed did monitoring agent succeed to connect to the node.
	MonitoringFailed bool `json:"monitoring_failed,omitempty"`
}
