package aggregate

import v3 "github.com/mysteriumnetwork/discovery/proposal/v3"

type Proposal struct {
	Meta           `json:",inline"`
	ProviderID     string            `json:"provider_id"`
	AccessPolicies []v3.AccessPolicy `json:"access_policies,omitempty"`

	Services []ProviderService `json:"services"`
}

type ProviderService struct {
	*Meta       `json:",inline,omitempty"`
	ServiceType string `json:"service_type"`
}

type Meta struct {
	Quality  *v3.Quality   `json:"quality,omitempty"`
	Location *v3.Location  `json:"location,omitempty"`
	Contacts *[]v3.Contact `json:"contacts,omitempty"`
}

func NewProposal(providerID string) *Proposal {
	return &Proposal{
		ProviderID: providerID,
	}
}

func (p *Proposal) mergeProposal(newProposal Proposal) {
	// Update provider's meta with the latest data
	p.Meta = newProposal.Meta

	for _, s := range newProposal.Services {
		p.addService(s)
	}
}

func (p *Proposal) addService(service ProviderService) {
	for _, s := range p.Services {
		if s.ServiceType == service.ServiceType {
			return
		}
	}
	p.Services = append(p.Services, ProviderService{
		ServiceType: service.ServiceType,
	})
}

func (p *Proposal) getService(serviceType string) *ProviderService {
	for _, s := range p.Services {
		if s.ServiceType == serviceType {
			if s.Meta.Contacts == nil {
				s.Meta.Contacts = p.Meta.Contacts
			}
			if s.Meta.Location == nil {
				s.Meta.Location = p.Meta.Location
			}
			if s.Meta.Quality == nil {
				s.Meta.Quality = p.Meta.Quality
			}
			return &s
		}
	}
	return nil
}
