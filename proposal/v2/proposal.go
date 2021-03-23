package v2

import (
	"encoding/json"
	"math/big"
)

const Format = "service-proposal/v2"

type Proposal struct {
	Format         string         `json:"format"`
	ProviderID     string         `json:"provider_id"`
	ServiceType    string         `json:"service_type"`
	Location       Location       `json:"location"`
	Price          Price          `json:"price"`
	Contacts       []Contact      `json:"contacts"`
	AccessPolicies []AccessPolicy `json:"access_policies,omitempty"`
}

func NewProposal(providerID, serviceType string) *Proposal {
	return &Proposal{
		Format:      Format,
		ProviderID:  providerID,
		ServiceType: serviceType,
	}
}

func ProposalProviderIDS(proposals []Proposal) []string {
	ids := make([]string, len(proposals))
	for idx, p := range proposals {
		ids[idx] = p.ProviderID
	}
	return ids
}

func (p Proposal) MarshalBinary() (data []byte, err error) {
	marshal, err := json.Marshal(p)
	return marshal, err
}

func (p Proposal) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &p)
}

type Location struct {
	Continent string `json:"continent,omitempty"`
	Country   string `json:"country,omitempty"`
	City      string `json:"city,omitempty"`
	ASN       int    `json:"asn,omitempty"`
	ISP       string `json:"isp,omitempty"`
	IPType    string `json:"ip_type,omitempty"`
}

type Price struct {
	Currency Currency `json:"currency"`
	PerHour  *big.Int `json:"per_hour"`
	PerGiB   *big.Int `json:"per_gib"`
}

type Contact struct {
	Type       string           `json:"type"`
	Definition *json.RawMessage `json:"definition"`
}

// AccessPolicy represents the access controls for proposal
type AccessPolicy struct {
	ID     string `json:"id"`
	Source string `json:"source"`
}

type Quality struct {
	ProviderID string  `json:"provider_id"`
	Quality    float32 `json:"quality"`
}

func (p Quality) MarshalBinary() (data []byte, err error) {
	marshal, err := json.Marshal(p)
	return marshal, err
}

func (p Quality) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &p)
}

func NewQuality(providerID string, quality float32) *Quality {
	return &Quality{ProviderID: providerID, Quality: quality}
}
