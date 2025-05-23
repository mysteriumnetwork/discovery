// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package proposal

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	v3 "github.com/mysteriumnetwork/discovery/proposal/v3"
	"github.com/mysteriumnetwork/discovery/quality/oracleapi"
)

var ErrProposalIncompatible = errors.New("compatibility too low")

type Enhancer interface {
	Enhance(proposal *v3.Proposal)
}

type Repository struct {
	expirationJobDelay           time.Duration
	expirationDuration           time.Duration
	mu                           sync.RWMutex
	proposals                    map[string]record
	proposalsHardLimitPerCountry int
	proposalsSoftLimitPerCountry int
	compatibilityMin             int
}

type repoListOpts struct {
	providerIDS        []string
	serviceType        string
	country            string
	ipType             string
	accessPolicy       string
	accessPolicySource string
	compatibilityMin   int
	compatibilityMax   int
}

type repoMetadataOpts struct {
	providerID string
}

type record struct {
	proposal  v3.Proposal
	expiresAt time.Time
}

func NewRepository(proposalsHardLimitPerCountry, proposalsSoftLimitPerCountry, compatibilityMin int) *Repository {
	return &Repository{
		expirationDuration:           3*time.Minute + 10*time.Second,
		expirationJobDelay:           20 * time.Second,
		proposals:                    make(map[string]record),
		proposalsHardLimitPerCountry: proposalsHardLimitPerCountry,
		proposalsSoftLimitPerCountry: proposalsSoftLimitPerCountry,
		compatibilityMin:             compatibilityMin,
	}
}

func (r *Repository) List(opts repoListOpts, limited bool) (res []v3.Proposal) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	proposals := r.proposals
	countryLimit := make(map[string]int)

	if len(opts.providerIDS) > 0 && opts.serviceType != "" {
		// short path: skip iteration over collection,
		// lookup specific entries in reduced collection
		// instead
		proposals = make(map[string]record)
		for _, reqProviderID := range opts.providerIDS {
			key := v3.Proposal{
				ProviderID:  reqProviderID,
				ServiceType: opts.serviceType,
			}.Key()
			if proposalFound, ok := r.proposals[key]; ok {
				proposals[key] = proposalFound
			}
		}
	}

	for _, p := range proposals {
		if !match(p.proposal, opts) {
			continue
		}

		countryLimit[p.proposal.Location.Country]++

		if !limited || countryLimit[p.proposal.Location.Country] <= r.proposalsHardLimitPerCountry {
			if !limited || countryLimit[p.proposal.Location.Country] <= r.proposalsSoftLimitPerCountry || countryLimit[p.proposal.Location.Country]%10 == 0 {
				res = append(res, p.proposal)
			}
		}
	}

	return res
}

func (r *Repository) ListCountriesNumbers(opts repoListOpts) map[string]int {
	res := make(map[string]int)

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.proposals {
		if !match(p.proposal, opts) {
			continue
		}

		res[p.proposal.Location.Country]++
	}

	return res
}

func (r *Repository) Metadata(opts repoMetadataOpts, or map[string]*oracleapi.DetailedQuality) (res []v3.Metadata) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.proposals {
		whitelisted := false
		monitoringFailed := false

		q, ok := or[p.proposal.Key()]
		if ok {
			monitoringFailed = q.MonitoringFailed
		}

		for _, v := range p.proposal.AccessPolicies {
			if v.ID == "mysterium" {
				whitelisted = true
			}
		}

		res = append(res, v3.Metadata{
			ProviderID:       p.proposal.ProviderID,
			ServiceType:      p.proposal.ServiceType,
			Country:          p.proposal.Location.Country,
			ISP:              p.proposal.Location.ISP,
			IPType:           (string)(p.proposal.Location.IPType),
			Whitelist:        whitelisted,
			MonitoringFailed: monitoringFailed,
			UpdatedAt:        p.expiresAt.Add(-r.expirationDuration),
		})
	}

	return res
}

func (r *Repository) Store(proposal v3.Proposal) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if proposal.Compatibility < r.compatibilityMin {
		return ErrProposalIncompatible
	}

	proposal = removeOldBrokerIPs(proposal)

	r.proposals[proposal.Key()] = record{
		proposal:  proposal,
		expiresAt: time.Now().Add(r.expirationDuration),
	}

	proposalAdded(proposal)

	return nil
}

func (r *Repository) Expire() (count int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var activeProposals []v3.Proposal
	for k, v := range r.proposals {
		if time.Now().After(v.expiresAt) {
			proposalExpired(r.proposals[k].proposal)
			delete(r.proposals, k)
			count++
		} else {
			activeProposals = append(activeProposals, v.proposal)
		}
	}

	proposalActive(activeProposals)

	return count
}

func removeOldBrokerIPs(proposal v3.Proposal) v3.Proposal {
	for i, c := range proposal.Contacts {
		if c.Type != "nats/p2p/v1" {
			continue
		}

		type definition struct {
			BrokerAddresses []string `json:"broker_addresses"`
		}

		var def definition
		if c.Definition == nil {
			continue
		}

		if err := json.Unmarshal(*c.Definition, &def); err != nil {
			continue
		}

		var addresses []string

		for _, addr := range def.BrokerAddresses {
			if strings.Contains(addr, "51.158.204.30") || strings.Contains(addr, "51.158.204.75") || strings.Contains(addr, "51.158.204.9") || strings.Contains(addr, "51.158.204.23") {
				continue
			}

			addresses = append(addresses, addr)
		}

		result, err := json.Marshal(addresses)
		if err != nil {
			continue
		}

		msg := json.RawMessage(`{"broker_addresses":` + string(result) + `}`)

		proposal.Contacts[i] = v3.Contact{
			Type:       c.Type,
			Definition: &msg,
		}
	}

	return proposal
}

func (r *Repository) Remove(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	proposalRemoved(r.proposals[key].proposal)
	delete(r.proposals, key)
}

func match(p v3.Proposal, opts repoListOpts) bool {
	if len(opts.providerIDS) > 0 {
		found := false

		for _, id := range opts.providerIDS {
			if p.ProviderID == id {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	if opts.serviceType != "" && p.ServiceType != opts.serviceType {
		return false
	}

	if opts.country != "" && p.Location.Country != opts.country {
		return false
	}

	if opts.compatibilityMin != 0 {
		if p.Compatibility < opts.compatibilityMin {
			return false
		}
	}

	if opts.compatibilityMax != 0 {
		if p.Compatibility > opts.compatibilityMax {
			return false
		}
	}

	if opts.ipType != "" && p.Location.IPType != v3.IPType(opts.ipType) {
		return false
	}

	if opts.accessPolicy != "" && opts.accessPolicy != "all" {
		found := false

		for _, v := range p.AccessPolicies {
			if v.ID == opts.accessPolicy {
				found = true
			}
		}

		if !found {
			return false
		}
	} else if opts.accessPolicy == "" && len(p.AccessPolicies) > 0 {
		return false
	}

	if opts.accessPolicySource != "" {
		found := false

		for _, v := range p.AccessPolicies {
			if v.Source == opts.accessPolicySource {
				found = true
			}
		}

		if !found {
			return false
		}
	}

	return true
}
