// Copyright (c) 2022 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package proposal

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	v3 "github.com/mysteriumnetwork/discovery/proposal/v3"
)

var discoveryProposalAdded = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "discovery_proposal_added",
		Help: "Service proposal added to the discovery",
	},
	[]string{"format", "compatibility", "service_type", "country", "access_policy", "node_type"},
)

var discoveryProposalExpired = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "discovery_proposal_expired",
		Help: "Service proposal expired in the discovery",
	},
	[]string{"format", "compatibility", "service_type", "country", "access_policy", "node_type"},
)

var discoveryProposalRemoved = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "discovery_proposal_removed",
		Help: "Service proposal removed from the discovery",
	},
	[]string{"format", "compatibility", "service_type", "country", "access_policy", "node_type"},
)

var discoveryProposalActive = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "discovery_proposal_active",
		Help: "Service proposal active in the discovery",
	},
	[]string{"format", "compatibility", "service_type", "country", "access_policy", "node_type"},
)

var discoveryProvidersTotal = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "discovery_providers_total",
		Help: "Total number of active providers in the discovery",
	},
	[]string{"format", "compatibility", "country", "node_type"},
)

var discoveryProviderASNDiversity = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "discovery_provider_asn_diversity",
		Help: "Number of providers sharing the same ASN as the given provider",
	},
	[]string{"provider", "asn", "country", "node_type"},
)

func init() {
	prometheus.MustRegister(discoveryProposalAdded, discoveryProposalExpired, discoveryProposalRemoved, discoveryProposalActive, discoveryProvidersTotal, discoveryProviderASNDiversity)
}

func accessPolicies(p v3.Proposal) string {
	var ap []string

	for _, policy := range p.AccessPolicies {
		ap = append(ap, policy.ID)
	}

	return strings.Join(ap, ",")
}

func proposalAdded(p v3.Proposal) {
	discoveryProposalAdded.WithLabelValues(p.Format, strconv.Itoa(p.Compatibility), p.ServiceType, p.Location.Country, accessPolicies(p), string(p.Location.IPType)).Add(1)
}

func proposalExpired(p v3.Proposal) {
	discoveryProposalExpired.WithLabelValues(p.Format, strconv.Itoa(p.Compatibility), p.ServiceType, p.Location.Country, accessPolicies(p), string(p.Location.IPType)).Add(1)
}

func proposalRemoved(p v3.Proposal) {
	discoveryProposalRemoved.WithLabelValues(p.Format, strconv.Itoa(p.Compatibility), p.ServiceType, p.Location.Country, accessPolicies(p), string(p.Location.IPType)).Add(1)
}

func proposalActive(proposals []v3.Proposal) {
	active := make(map[string]int)
	total := make(map[string]int)
	providers := make(map[string]struct{})

	for _, p := range proposals {
		key := strings.Join([]string{p.Format, strconv.Itoa(p.Compatibility), p.ServiceType, p.Location.Country, accessPolicies(p), string(p.Location.IPType)}, "|")
		active[key]++

		if _, ok := providers[p.ProviderID]; !ok {
			keyTotal := strings.Join([]string{p.Format, strconv.Itoa(p.Compatibility), p.Location.Country, string(p.Location.IPType)}, "|")
			total[keyTotal]++
			providers[p.ProviderID] = struct{}{}
		}

	}

	for labels, value := range active {
		discoveryProposalActive.WithLabelValues(strings.Split(labels, "|")...).Set(float64(value))
	}

	for labels, value := range total {
		discoveryProvidersTotal.WithLabelValues(strings.Split(labels, "|")...).Set(float64(value))
	}

	providerASNDiversity(proposals)
}

type providerInfo struct {
	asn      int
	country  string
	nodeType string
}

func providerASNDiversity(proposals []v3.Proposal) {
	providers := make(map[string]providerInfo)
	asnCount := make(map[int]int)
	for _, p := range proposals {
		if _, ok := providers[p.ProviderID]; !ok {
			providers[p.ProviderID] = providerInfo{
				asn:      p.Location.ASN,
				country:  p.Location.Country,
				nodeType: string(p.Location.IPType),
			}
			asnCount[p.Location.ASN]++
		}
	}

	// Reset stale label combinations before writing fresh values.
	discoveryProviderASNDiversity.Reset()
	for providerID, info := range providers {
		discoveryProviderASNDiversity.WithLabelValues(providerID, strconv.Itoa(info.asn), info.country, info.nodeType).Set(float64(asnCount[info.asn]))
	}
}
