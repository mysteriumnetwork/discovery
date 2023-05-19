package v3_test

import (
	"encoding/json"
	v3 "github.com/mysteriumnetwork/discovery/proposal/v3"
	"reflect"
	"testing"
)

func TestProposal_MarshalBinary(t *testing.T) {
	proposalJson := `{"id":0,"format":"service-proposal/v3","compatibility":2,"provider_id":"0x55d9995cf3482ee0628fe25e3c95a95899f23cad","service_type":"scraping","location":{"continent":"EU","country":"PL","region":"Malopolskie","city":"Nowy Sacz","asn":5617,"isp":"Orange Polska Spolka Akcyjna","ip_type":"residential"},"contacts":[{"type":"nats/p2p/v1","definition":{"broker_addresses":["nats://broker.mysterium.network:4222","nats://broker.mysterium.network:4222","nats://51.15.22.197:4222","nats://51.15.23.12:4222","nats://51.15.23.14:4222","nats://51.15.23.16:4222"]}}],"access_policies":[{"id":"mysterium","source":"https://trust.mysterium.network/api/v1/access-policies/mysterium"}],"quality":{"quality":2.0250000000000004,"latency":824.898867,"bandwidth":10.275812,"uptime":24}}`
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

// before
//go test github.com/mysteriumnetwork/discovery/proposal/v3 -bench=. -run BenchmarkProposal_UnmarshalBinary -count 10
//goos: darwin
//goarch: amd64
//pkg: github.com/mysteriumnetwork/discovery/proposal/v3
//cpu: Intel(R) Core(TM) i7-4770HQ CPU @ 2.20GHz
//BenchmarkProposal_UnmarshalBinary-8   	      12	  98144248 ns/op	24918711 B/op	       8 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	      12	  95996542 ns/op	24918710 B/op	       7 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	      12	  94519130 ns/op	19325925 B/op	       4 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	      12	  95355681 ns/op	24918693 B/op	       7 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	      12	  95350196 ns/op	24918740 B/op	       8 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	      12	  98040917 ns/op	24918686 B/op	       7 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	      12	  95320773 ns/op	24918732 B/op	       8 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	      12	  96656234 ns/op	24918685 B/op	       7 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	      12	  95178269 ns/op	19325925 B/op	       4 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	      12	  95749498 ns/op	24918567 B/op	       8 allocs/op
//PASS
//ok  	github.com/mysteriumnetwork/discovery/proposal/v3	19.150s

// after
//count 10
//goos: darwin
//goarch: amd64
//pkg: github.com/mysteriumnetwork/discovery/proposal/v3
//cpu: Intel(R) Core(TM) i7-4770HQ CPU @ 2.20GHz
//BenchmarkProposal_UnmarshalBinary-8   	      24	  49096402 ns/op	103143621 B/op	  330002 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	      22	  47942055 ns/op	103143645 B/op	  330002 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	      22	  47024309 ns/op	103143623 B/op	  330002 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	      24	  47138722 ns/op	103143608 B/op	  330002 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	      24	  47355233 ns/op	103143637 B/op	  330002 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	      22	  46616128 ns/op	103143642 B/op	  330002 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	      22	  48310278 ns/op	103143592 B/op	  330002 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	      24	  48881929 ns/op	103143624 B/op	  330002 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	      21	  67595264 ns/op	103143594 B/op	  330002 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	      25	  48716040 ns/op	103143609 B/op	  330002 allocs/op
//PASS
//ok  	github.com/mysteriumnetwork/discovery/proposal/v3	14.768s
//
//goos: darwin
//goarch: amd64
//pkg: github.com/mysteriumnetwork/discovery/proposal/v3
//cpu: Intel(R) Core(TM) i7-4770HQ CPU @ 2.20GHz
//BenchmarkProposal_UnmarshalBinary-8   	  145886	      7047 ns/op	    2298 B/op	       7 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	  162248	      7112 ns/op	    2298 B/op	       7 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	  164491	      7566 ns/op	    2298 B/op	       7 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	  163942	      7107 ns/op	    2298 B/op	       7 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	  163242	      7102 ns/op	    2298 B/op	       7 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	  159690	      7135 ns/op	    2298 B/op	       7 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	  161736	      6998 ns/op	    2298 B/op	       7 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	  162242	      7168 ns/op	    2298 B/op	       7 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	  152942	      6927 ns/op	    2298 B/op	       7 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	  158584	      6997 ns/op	    2298 B/op	       7 allocs/op
//PASS
//ok  	github.com/mysteriumnetwork/discovery/proposal/v3	12.466s
