// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package proposal

import (
	"strings"
	"sync"
	"time"

	v3 "github.com/mysteriumnetwork/discovery/proposal/v3"
	"github.com/mysteriumnetwork/discovery/quality/oracleapi"
)

const (
	// TODO: lower this values once dvpn apps start using proposal numbers endpoint.
	countryHardLimit = 1000
	countrySoftLimit = 100
)

type Enhancer interface {
	Enhance(proposal *v3.Proposal)
}

type Repository struct {
	expirationJobDelay time.Duration
	expirationDuration time.Duration
	mu                 sync.RWMutex
	proposals          map[string]record
	enhancers          []Enhancer
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
	tags               string
}

type repoMetadataOpts struct {
	providerID string
}

type record struct {
	proposal  v3.Proposal
	expiresAt time.Time
}

func NewRepository(enhancers []Enhancer) *Repository {
	return &Repository{
		expirationDuration: 3*time.Minute + 10*time.Second,
		expirationJobDelay: 20 * time.Second,
		proposals:          make(map[string]record),
		enhancers:          enhancers,
	}
}

func (r *Repository) List(opts repoListOpts) (res []v3.Proposal) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	countryLimit := make(map[string]int)

	for _, p := range r.proposals {
		if !match(p.proposal, opts) {
			continue
		}

		countryLimit[p.proposal.Location.Country]++

		if countryLimit[p.proposal.Location.Country] <= countryHardLimit {
			if countryLimit[p.proposal.Location.Country] <= countrySoftLimit || countryLimit[p.proposal.Location.Country]%10 == 0 {
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

	r.proposals[proposal.Key()] = record{
		proposal:  proposal,
		expiresAt: time.Now().Add(r.expirationDuration),
	}

	return nil
}

func (r *Repository) Expire() (count int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for k, v := range r.proposals {
		if time.Now().After(v.expiresAt) {
			delete(r.proposals, k)
			count++
		}
	}

	return count
}

func (r *Repository) Remove(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()

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

	if opts.compatibilityMin != 0 || opts.compatibilityMax != 0 {
		if opts.compatibilityMin > p.Compatibility && p.Compatibility < opts.compatibilityMax {
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

	if opts.tags != "" {
		found := false

		for _, ot := range strings.Split(opts.tags, ",") {
			for _, pt := range p.Tags {
				if ot == pt {
					found = true
				}
			}
		}

		if !found {
			return false
		}
	}

	return true
}
