package e2e

import (
	"encoding/json"
	"math/big"

	v2 "github.com/mysteriumnetwork/discovery/proposal/v2"
)

type template struct {
	proposalMessage *v2.ProposalPingMessage
	quality         v2.Quality
}

func newTemplate() *template {
	t := proposalTemplate // make a copy
	return &template{proposalMessage: &t}
}

func (t *template) prices(perGiB, perHour int64) *template {
	t.proposalMessage.Proposal.Price.PerGiB = new(big.Int).SetInt64(perGiB)
	t.proposalMessage.Proposal.Price.PerHour = new(big.Int).SetInt64(perHour)
	return t
}

func (t *template) providerID(id string) *template {
	t.proposalMessage.Proposal.ProviderID = id
	return t
}

func (t *template) serviceType(serviceType string) *template {
	t.proposalMessage.Proposal.ServiceType = serviceType
	return t
}

func (t *template) country(country string) *template {
	t.proposalMessage.Proposal.Location.Country = country
	return t
}

func (t *template) compatibility(c int) *template {
	t.proposalMessage.Proposal.Compatibility = c
	return t
}

func (t *template) publishPing() error {
	err := defaultBroker.PublishPingOneV2(*t.proposalMessage)
	if err != nil {
		return err
	}

	return nil
}

func (t *template) unregister() error {
	err := defaultBroker.PublishUnregisterOneV2(v2.ProposalUnregisterMessage{
		Proposal: t.proposalMessage.Proposal,
	})
	if err != nil {
		return err
	}

	return nil
}

var proposalTemplate = v2.ProposalPingMessage{
	Proposal: v2.Proposal{
		Format:        "service-proposal/v2",
		Compatibility: 0,
		ProviderID:    "0xfa7855e183c3474eddd9d3a0088d2b1abddde837",
		ServiceType:   "openvpn",
		Location: v2.Location{
			ASN:       6871,
			ISP:       "Plusnet",
			City:      "England",
			Country:   "GB",
			IPType:    "residential",
			Continent: "EU",
		},
		Price: v2.Price{
			Currency: "MYSTT",
			PerHour:  new(big.Int).SetInt64(500000000000000),
			PerGiB:   new(big.Int).SetInt64(500000000000000),
		},
		Contacts: []v2.Contact{
			{
				Type:       "nats/p2p/v1",
				Definition: definition(),
			},
		},
		AccessPolicies: nil,
	},
}

func definition() *json.RawMessage {
	def := json.RawMessage{}
	def = []byte("{\"broker_addresses\":[\"nats://testnet2-broker.mysterium.network:4222\",\"nats://testnet2-broker.mysterium.network:4222\",\"nats://95.216.204.232:4222\"]}")
	return &def
}
