// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package v1

import (
	"encoding/json"
	"math/big"
	"reflect"
	"time"

	v2 "github.com/mysteriumnetwork/discovery/proposal/v2"
)

type ProposalPingMessage struct {
	Proposal ServiceProposal `json:"proposal"`
}

func (p ProposalPingMessage) IsEmpty() bool {
	return reflect.DeepEqual(p, ProposalPingMessage{})
}

type ProposalUnregisterMessage struct {
	Proposal ServiceProposal `json:"proposal"`
}

func (p ProposalUnregisterMessage) IsEmpty() bool {
	return reflect.DeepEqual(p, ProposalPingMessage{})
}

func (p ProposalUnregisterMessage) Key() string {
	return p.Proposal.ServiceType + ":" + p.Proposal.ProviderID
}

type ServiceProposal struct {
	// Per provider unique serial number of service description provided
	// TODO Not supported yet
	ID int `json:"id"`

	// A version number is included in the proposal to allow extensions to the proposal format
	Format string `json:"format"`

	// Type of service type offered
	ServiceType string `json:"service_type"`

	// Qualitative service definition
	ServiceDefinition ServiceDefinition `json:"service_definition"`

	// Type of service payment method
	PaymentMethodType string `json:"payment_method_type"`

	// Service payment & usage metering definition
	PaymentMethod PaymentMethod `json:"payment_method"`

	// Unique identifier of a provider
	ProviderID string `json:"provider_id"`

	// Communication methods possible
	ProviderContacts []v2.Contact `json:"provider_contacts"`

	// AccessPolicies represents the access controls for proposal
	AccessPolicies []v2.AccessPolicy `json:"access_policies,omitempty"`
}

func (s *ServiceProposal) ConvertToV2() *v2.Proposal {
	p := v2.NewProposal(s.ProviderID, s.ServiceType)
	loc := s.ServiceDefinition.Location
	p.Location = v2.Location{
		Continent: loc.Continent,
		Country:   loc.Country,
		City:      loc.City,
		ASN:       loc.ASN,
		ISP:       loc.ISP,
		IPType:    v2.IPType(loc.NodeType),
	}

	p.Price = v2.Price{
		Currency: v2.Currency(s.PaymentMethod.Price.Currency),
		PerHour:  s.PaymentMethod.pricePerHour(),
		PerGiB:   s.PaymentMethod.pricePerGiB(),
	}

	p.Contacts = s.ProviderContacts
	p.AccessPolicies = s.AccessPolicies

	return p
}

func (s *ServiceProposal) MarshalBinary() (data []byte, err error) {
	return json.Marshal(s)
}

func (s *ServiceProposal) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &s)
}

// ServiceDefinition interface is interface for all service definition types
type ServiceDefinition struct {
	Location Location `json:"location"`
}

// Location struct represents geographic location of service provider
type Location struct {
	Continent string `json:"continent,omitempty"`
	Country   string `json:"country,omitempty"`
	City      string `json:"city,omitempty"`

	ASN      int    `json:"asn,omitempty"`
	ISP      string `json:"isp,omitempty"`
	NodeType string `json:"node_type,omitempty"`
}

// Money holds the currency type and amount
type Money struct {
	Amount   *big.Int `json:"amount,omitempty"`
	Currency Currency `json:"currency,omitempty"`
}

// Currency represents a supported currency
type Currency string

// PaymentRate represents the payment rate
type PaymentRate struct {
	PerTime time.Duration
	PerByte uint64
}
