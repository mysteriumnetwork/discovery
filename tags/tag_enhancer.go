package tags

import (
	v3 "github.com/mysteriumnetwork/discovery/proposal/v3"
	"github.com/rs/zerolog/log"
)

type tagAPI interface {
	GetTags(providerID string) ([]string, error)
}

type Enhancer struct {
	tagAPI tagAPI
}

func NewEnhancer(tagAPI tagAPI) *Enhancer {
	return &Enhancer{
		tagAPI: tagAPI,
	}
}

func (e *Enhancer) Enhance(proposal *v3.Proposal) {
	tags, err := e.tagAPI.GetTags(proposal.ProviderID)
	if err != nil {
		log.Error().Err(err).Str("providerID", proposal.ProviderID).Msg("could not get tags")
		return
	}

	proposal.Tags = tags
}
