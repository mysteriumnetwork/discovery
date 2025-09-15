// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package aggregate

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	v3 "github.com/mysteriumnetwork/discovery/proposal/v3"
)

type Repository struct {
	expirationJobDelay           time.Duration
	expirationDuration           time.Duration
	mu                           sync.RWMutex
	proposals                    map[string]record
	proposalsHardLimitPerCountry int
	proposalsSoftLimitPerCountry int
}

type RepoListOpts struct {
	ProviderIDS  []string
	ServiceType  string
	Country      string
	IpType       string
	AccessPolicy string
}

type record struct {
	proposal  Proposal
	expiresAt time.Time
}

func NewRepository(proposalsHardLimitPerCountry, proposalsSoftLimitPerCountry int) *Repository {
	return &Repository{
		expirationDuration:           3*time.Minute + 10*time.Second,
		expirationJobDelay:           20 * time.Second,
		proposals:                    make(map[string]record),
		proposalsHardLimitPerCountry: proposalsHardLimitPerCountry,
		proposalsSoftLimitPerCountry: proposalsSoftLimitPerCountry,
	}
}

func (r *Repository) List(opts RepoListOpts) (res []Proposal) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	proposals := r.proposals

	if len(opts.ProviderIDS) > 0 && opts.ServiceType != "" {
		// short path: skip iteration over collection,
		// lookup specific entries in reduced collection
		// instead
		proposals = make(map[string]record)
		for _, reqProviderID := range opts.ProviderIDS {
			if proposalFound, ok := r.proposals[reqProviderID]; ok {
				proposals[reqProviderID] = proposalFound
			}
		}
	}

	for _, p := range proposals {
		if !match(p.proposal, opts) {
			continue
		}
		res = append(res, p.proposal)
	}

	return res
}

func (r *Repository) StoreV3(proposalV3 v3.Proposal) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	proposal := removeOldBrokerIPs(fromV3(proposalV3))

	if existing, ok := r.proposals[proposal.ProviderID]; ok {
		existing.proposal.mergeProposal(proposal)
		r.proposals[proposal.ProviderID] = record{
			proposal:  existing.proposal,
			expiresAt: time.Now().Add(r.expirationDuration),
		}
		return nil
	}

	r.proposals[proposal.ProviderID] = record{
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

func removeOldBrokerIPs(proposal Proposal) Proposal {
	for i, c := range *proposal.Contacts {
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

		(*proposal.Contacts)[i] = v3.Contact{
			Type:       c.Type,
			Definition: &msg,
		}
	}

	return proposal
}

func (r *Repository) Remove(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.proposals, key)
}

func match(p Proposal, opts RepoListOpts) bool {
	if len(opts.ProviderIDS) > 0 {
		found := false

		for _, id := range opts.ProviderIDS {
			if p.ProviderID == id {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	service := &ProviderService{}
	if opts.ServiceType != "" {
		service = p.getService(opts.ServiceType)
		if service == nil {
			return false
		}
	}

	if opts.Country != "" && service.Location.Country != opts.Country {
		return false
	}

	if opts.IpType != "" && service.Location.IPType != v3.IPType(opts.IpType) {
		return false
	}

	if opts.AccessPolicy != "" && opts.AccessPolicy != "all" {
		found := false

		for _, v := range p.AccessPolicies {
			if v.ID == opts.AccessPolicy {
				found = true
			}
		}

		if !found {
			return false
		}
	} else if opts.AccessPolicy == "" && len(p.AccessPolicies) > 0 {
		return false
	}

	return true
}

func fromV3(p v3.Proposal) Proposal {
	return Proposal{
		ProviderID:     p.ProviderID,
		AccessPolicies: p.AccessPolicies,
		Meta: Meta{
			Quality:  &p.Quality,
			Location: &p.Location,
			Contacts: &p.Contacts,
		},
		Services: []ProviderService{
			{
				ServiceType: p.ServiceType,
			},
		},
	}
}
