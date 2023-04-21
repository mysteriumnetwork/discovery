package v3_test

import (
	"encoding/json"
	v3 "github.com/mysteriumnetwork/discovery/proposal/v3"
	"testing"
)

func BenchmarkProposal_UnmarshalBinary(b *testing.B) {
	b.ReportAllocs()
	proposalsN := 30000
	proposals := make(v3.Proposals, proposalsN)
	def := json.RawMessage(`{"broker_addresses": ["nats://broker.mysterium.network:4222","nats://broker.mysterium.network:4222","nats://51.15.22.197:4222","nats://51.15.23.12:4222","nats://51.15.23.14:4222","nats://51.15.23.16:4222"]}`)
	proposal := &v3.Proposal{
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
		_, err := proposals.MarshalJ()
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
//go test github.com/mysteriumnetwork/discovery/proposal/v3 -bench=. -run BenchmarkProposal_UnmarshalBinary -count 5
//goos: darwin
//goarch: amd64
//pkg: github.com/mysteriumnetwork/discovery/proposal/v3
//cpu: Intel(R) Core(TM) i7-4770HQ CPU @ 2.20GHz
//BenchmarkProposal_UnmarshalBinary-8   	      25	  48259626 ns/op	93639888 B/op	  330002 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	      24	  46120996 ns/op	93639860 B/op	  330002 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	      22	  46043380 ns/op	93639906 B/op	  330003 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	      22	  46046696 ns/op	93639886 B/op	  330002 allocs/op
//BenchmarkProposal_UnmarshalBinary-8   	      24	  46295770 ns/op	93639912 B/op	  330003 allocs/op
//PASS
//ok  	github.com/mysteriumnetwork/discovery/proposal/v3	6.628s
