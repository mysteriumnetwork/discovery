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
	[]string{"format", "compatibility", "service_type", "country", "access_policy"},
)

var discoveryProposalExpired = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "discovery_proposal_expired",
		Help: "Service proposal expired in the discovery",
	},
	[]string{"format", "compatibility", "service_type", "country", "access_policy"},
)

var discoveryProposalRemoved = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "discovery_proposal_removed",
		Help: "Service proposal removed from the discovery",
	},
	[]string{"format", "compatibility", "service_type", "country", "access_policy"},
)

var discoveryProposalActive = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "discovery_proposal_active",
		Help: "Service proposal active in the discovery",
	},
	[]string{"format", "compatibility", "service_type", "country", "access_policy"},
)

func init() {
	prometheus.MustRegister(discoveryProposalAdded, discoveryProposalExpired, discoveryProposalRemoved, discoveryProposalActive)
}

func accessPolicies(p v3.Proposal) string {
	var ap []string

	for _, policy := range p.AccessPolicies {
		ap = append(ap, policy.ID)
	}

	return strings.Join(ap, ",")
}

func proposalAdded(p v3.Proposal) {
	discoveryProposalAdded.WithLabelValues(p.Format, strconv.Itoa(p.Compatibility), p.ServiceType, p.Location.Country, accessPolicies(p)).Add(1)
}

func proposalExpired(p v3.Proposal) {
	discoveryProposalExpired.WithLabelValues(p.Format, strconv.Itoa(p.Compatibility), p.ServiceType, p.Location.Country, accessPolicies(p)).Add(1)
}

func proposalRemoved(p v3.Proposal) {
	discoveryProposalRemoved.WithLabelValues(p.Format, strconv.Itoa(p.Compatibility), p.ServiceType, p.Location.Country, accessPolicies(p)).Add(1)
}

func proposalActive(proposals []v3.Proposal) {
	m := make(map[string]int)

	for _, p := range proposals {
		key := strings.Join([]string{p.Format, strconv.Itoa(p.Compatibility), p.ServiceType, p.Location.Country, accessPolicies(p)}, "|")
		m[key]++
	}

	for labels, value := range m {
		discoveryProposalActive.WithLabelValues(strings.Split(labels, "|")...).Set(float64(value))
	}
}
