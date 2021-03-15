package v1

import (
	"math/big"
	"time"
)

type ProposalPingMessage struct {
	Proposal ServiceProposal `json:"proposal"`
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
	//ProviderContacts ContactList `json:"provider_contacts"`

	// AccessPolicies represents the access controls for proposal
	AccessPolicies *[]AccessPolicy `json:"access_policies,omitempty"`
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

// PaymentMethod is a interface for all types of payment methods
type PaymentMethod struct {
	Price    Money  `json:"price"`
	Duration int    `json:"duration"`
	Bytes    int    `json:"bytes"`
	Type     string `json:"type"`
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

// AccessPolicy represents the access controls for proposal
type AccessPolicy struct {
	ID     string `json:"id"`
	Source string `json:"source"`
}
