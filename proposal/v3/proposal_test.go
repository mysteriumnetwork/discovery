package v3_test

import (
	"encoding/json"
	"reflect"
	"testing"

	v3 "github.com/mysteriumnetwork/discovery/proposal/v3"
)

func TestProposal_MarshalBinary(t *testing.T) {
	proposalJson := `{"id":0,"format":"service-proposal/v3","compatibility":2,"provider_id":"0x55d9995cf3482ee0628fe25e3c95a95899f23cad","service_type":"scraping","location":{"continent":"EU","country":"PL","region":"Malopolskie","city":"Nowy Sacz","asn":5617,"isp":"Orange Polska Spolka Akcyjna","ip_type":"residential"},"contacts":[{"type":"nats/p2p/v1","definition":{"broker_addresses":["nats://broker.mysterium.network:4222","nats://broker.mysterium.network:4222","nats://51.15.22.197:4222","nats://51.15.23.12:4222","nats://51.15.23.14:4222","nats://51.15.23.16:4222"]}}],"access_policies":[{"id":"mysterium","source":"https://trust.mysterium.network/api/v1/access-policies/mysterium"}],"quality":{"quality":2.0250000000000004,"latency":824.898867,"bandwidth":10.275812,"uptime":24,"packetLoss":0.5}}`
	def := json.RawMessage(`{
			"broker_addresses": [
				"nats://broker.mysterium.network:4222",
				"nats://broker.mysterium.network:4222",
				"nats://51.15.22.197:4222",
				"nats://51.15.23.12:4222",
				"nats://51.15.23.14:4222",
				"nats://51.15.23.16:4222"
			]
	}`)
	actualProposal := v3.Proposal{
		ID:            0,
		Format:        "service-proposal/v3",
		Compatibility: 2,
		ProviderID:    "0x55d9995cf3482ee0628fe25e3c95a95899f23cad",
		ServiceType:   "scraping",
		Location: v3.Location{
			Continent: "EU",
			Country:   "PL",
			Region:    "Malopolskie",
			City:      "Nowy Sacz",
			ASN:       5617,
			ISP:       "Orange Polska Spolka Akcyjna",
			IPType:    "residential",
		},
		Contacts: []v3.Contact{
			{
				Type:       "nats/p2p/v1",
				Definition: &def,
			},
		},
		AccessPolicies: []v3.AccessPolicy{
			{
				ID:     "mysterium",
				Source: "https://trust.mysterium.network/api/v1/access-policies/mysterium",
			},
		},
		Quality: v3.Quality{
			Quality:          2.0250000000000004,
			Latency:          824.898867,
			Bandwidth:        10.275812,
			Uptime:           24,
			PacketLoss:       0.5,
			MonitoringFailed: false,
		},
	}

	data, err := actualProposal.MarshalBinary()
	if err != nil {
		t.Fatalf("Failed to marshal Proposal: %v", err)
	}

	// Compare the two Proposals
	if !reflect.DeepEqual(string(data), proposalJson) {
		t.Errorf("The expected and actual Proposals do not match.\nExpected: %v\nActual: %v", proposalJson, string(data))
	}
}

func BenchmarkProposal_UnmarshalBinary(b *testing.B) {
	b.ReportAllocs()
	proposalsN := 30000
	proposals := make([]v3.Proposal, proposalsN)
	def := json.RawMessage(`{"broker_addresses": ["nats://broker.mysterium.network:4222","nats://broker.mysterium.network:4222","nats://51.15.22.197:4222","nats://51.15.23.12:4222","nats://51.15.23.14:4222","nats://51.15.23.16:4222"]}`)
	proposal := v3.Proposal{
		ID:            1,
		Format:        "service-proposal/v3",
		Compatibility: 2,
		ProviderID:    "0x786de5d4370a524b0b430e831a18fc60c8b4754e",
		ServiceType:   "wireguard",
		Location: v3.Location{
			Continent: "NA",
			Country:   "US",
			Region:    "Florida",
			City:      "Miami",
			ASN:       1234,
			ISP:       "AT&T Internet Services",
			IPType:    "residential",
		},
		Contacts: []v3.Contact{
			{
				Type:       "nats/p2p/v1",
				Definition: &def,
			},
		},
		Quality: v3.Quality{
			Quality:          1.6500000000000001,
			Latency:          497.234331,
			Bandwidth:        1.825416,
			Uptime:           24,
			PacketLoss:       0.5,
			MonitoringFailed: true,
		},
	}
	for i := 0; i < proposalsN; i++ {
		proposals[i] = proposal
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(proposal)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.StopTimer()
}
