package e2e

import (
	"encoding/json"

	v3 "github.com/mysteriumnetwork/discovery/proposal/v3"
)

type template struct {
	proposalMessage *v3.ProposalPingMessage
	quality         v3.Quality
}

func newTemplate() *template {
	t := proposalTemplate // make a copy
	return &template{proposalMessage: &t}
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
	err := defaultBroker.PublishUnregisterOneV2(v3.ProposalUnregisterMessage{
		Proposal: t.proposalMessage.Proposal,
	})
	if err != nil {
		return err
	}

	return nil
}

var proposalTemplate = v3.ProposalPingMessage{
	Proposal: v3.Proposal{
		Format:        "service-proposal/v2",
		Compatibility: 0,
		ProviderID:    "0xfa7855e183c3474eddd9d3a0088d2b1abddde837",
		ServiceType:   "openvpn",
		Location: v3.Location{
			ASN:       6871,
			ISP:       "Plusnet",
			City:      "England",
			Country:   "GB",
			IPType:    "residential",
			Continent: "EU",
		},
		Contacts: []v3.Contact{
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
