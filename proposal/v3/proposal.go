// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package v3

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
)

const minProposalLength = 900

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

type Proposals []*Proposal

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

func (p *Proposal) Key() string {
	return p.ProviderID + "." + p.ServiceType
}

func (p *Proposal) MarshalBinary() (data []byte, err error) {
	marshal, err := json.Marshal(p)
	return marshal, err
}

func (p *Proposal) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}

func (p *Proposal) MarshalJ() ([]byte, error) {
	res := strings.Builder{}
	res.Grow(minProposalLength)
	res.WriteString(`{"format":"`)
	res.WriteString(p.Format)
	res.WriteString(`",`)

	res.WriteString(`"compatibility":`)
	res.WriteString(strconv.Itoa(p.Compatibility))
	res.WriteString(`,`)

	res.WriteString(`"provider_id":"`)
	res.WriteString(p.ProviderID)
	res.WriteString(`",`)

	res.WriteString(`"service_type":"`)
	res.WriteString(p.ServiceType)
	res.WriteString(`",`)

	res.WriteString(`"location":`)
	res.WriteString(`{`)
	res.WriteString(`"continent":"`)
	res.WriteString(p.Location.Continent)
	res.WriteString(`",`)
	res.WriteString(`"country":"`)
	res.WriteString(p.Location.Country)
	res.WriteString(`",`)
	res.WriteString(`"region":"`)
	res.WriteString(p.Location.Region)
	res.WriteString(`",`)
	res.WriteString(`"city":"`)
	res.WriteString(p.Location.City)
	res.WriteString(`",`)
	res.WriteString(`"asn":`)
	res.WriteString(strconv.Itoa(p.Location.ASN))
	res.WriteString(`,`)
	res.WriteString(`"isp":"`)
	res.WriteString(p.Location.ISP)
	res.WriteString(`",`)
	res.WriteString(`"ip_type":"`)
	res.WriteString(string(p.Location.IPType))
	res.WriteString(`"`)
	res.WriteString(`}`)

	res.WriteString(`,"quality":`)
	res.WriteString(`{`)
	res.WriteString(`"quality":`)
	res.WriteString("0")
	//res.WriteString(fmt.Sprintf("%f", p.Quality.Quality))
	res.WriteString(strconv.FormatFloat(p.Quality.Quality, 'f', -1, 64))
	res.WriteString(`,`)
	res.WriteString(`"latency":`)
	res.WriteString(strconv.FormatFloat(p.Quality.Latency, 'f', -1, 64))
	res.WriteString(`,`)
	res.WriteString(`"bandwidth":`)
	res.WriteString(strconv.FormatFloat(p.Quality.Bandwidth, 'f', -1, 64))
	res.WriteString(`,`)
	res.WriteString(`"uptime":`)
	res.WriteString(strconv.FormatFloat(p.Quality.Uptime, 'f', -1, 64))
	if p.Quality.MonitoringFailed {
		res.WriteString(`,`)
		res.WriteString(`"monitoring_failed":true`)
	}

	if p.Contacts != nil {
		res.WriteString(`,"contacts":[`)
		for i, c := range p.Contacts {
			res.WriteString(`{`)
			res.WriteString(`"type":"`)
			res.WriteString(c.Type)
			res.WriteString(`",`)
			res.WriteString(`"definition":`)
			res.Write(*c.Definition)
			res.WriteString(`}`)
			if i < len(p.Contacts)-1 {
				res.WriteString(`,`)
			}
		}
		res.WriteString(`]`)
	}
	if p.Tags != nil {
		res.WriteString(`,"tags":[`)
		for i, t := range p.Tags {
			res.WriteString(`"`)
			res.WriteString(t)
			res.WriteString(`"`)
			if i < len(p.Tags)-1 {
				res.WriteString(`,`)
			}
		}
		res.WriteString(`]`)
	}
	if p.AccessPolicies != nil {
		res.WriteString(`,"access_policies":[`)
		for i, ap := range p.AccessPolicies {
			res.WriteString(`{`)
			res.WriteString(`"id":"`)
			res.WriteString(ap.ID)
			res.WriteString(`",`)
			res.WriteString(`"source":"`)
			res.WriteString(ap.Source)
			res.WriteString(`"}`)
			if i < len(p.AccessPolicies)-1 {
				res.WriteString(`,`)
			}
		}
		res.WriteString(`]`)
	}

	res.WriteString(`}`)

	return []byte(res.String()), nil
}

func (ps Proposals) MarshalJ() ([]byte, error) {
	res := strings.Builder{}
	res.Grow(minProposalLength * len(ps))
	res.WriteString("[")
	for i, p := range ps {
		pb, err := p.MarshalJ()
		if err != nil {
			return nil, err
		}
		res.Write(pb)
		if i < len(ps)-1 {
			res.WriteString(",")
		}
	}
	res.WriteString("]")
	return []byte(res.String()), nil
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
	Region    string `json:"region,omitempty"`
	City      string `json:"city,omitempty"`
	ASN       int    `json:"asn,omitempty"`
	ISP       string `json:"isp,omitempty"`
	IPType    IPType `json:"ip_type,omitempty"`
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
	// Uptime in hours per day
	Uptime float64 `json:"uptime"`
	// MonitoringFailed did monitoring agent succeed to connect to the node.
	MonitoringFailed bool `json:"monitoring_failed,omitempty"`
}
