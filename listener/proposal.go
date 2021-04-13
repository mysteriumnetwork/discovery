// Copyright (c) 2021 BlockDev AG
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package listener

import (
	"encoding/json"
	"errors"

	"github.com/mysteriumnetwork/discovery/proposal"
	v1 "github.com/mysteriumnetwork/discovery/proposal/v1"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

type Listener struct {
	repository *proposal.Repository
	brokerURL  string
	conn       *nats.Conn
}

func New(brokerURL string, repository *proposal.Repository) *Listener {
	return &Listener{
		repository: repository,
		brokerURL:  brokerURL,
	}
}

func (l *Listener) Listen() error {
	conn, err := nats.Connect(l.brokerURL)
	if err != nil {
		return err
	}
	log.Info().Msgf("Connected to broker")
	l.conn = conn

	if _, err := conn.Subscribe("*.proposal-ping", func(msg *nats.Msg) {
		//log.Info().Msgf("Received a message [%s] %s", msg.Subject, string(msg.Data))
		pingMsg := v1.ProposalPingMessage{}
		if err := json.Unmarshal(msg.Data, &pingMsg); err != nil {
			log.Err(err).Msg("Failed to parse proposal")
		} else if pingMsg.IsEmpty() {
			log.Err(errors.New("unknown format")).
				Bytes("message", msg.Data).
				Msg("Failed to parse proposal")
		} else {
			p := pingMsg.Proposal.ConvertToV2()
			err := l.repository.Store(*p)
			if err != nil {
				log.Err(err).Msg("Failed to store proposal")
			}
		}
	}); err != nil {
		return err
	}

	return nil
}

func (l *Listener) Shutdown() {
	log.Info().Msg("Shutting down broker listener")
	l.conn.Close()
}
