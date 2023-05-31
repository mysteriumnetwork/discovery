package e2e

import (
	"encoding/json"
	v3 "github.com/mysteriumnetwork/discovery/proposal/v3"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

var defaultBroker = initBroker()

func initBroker() *Broker {
	broker, err := NewBroker(BrokerURL)
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize broker")
	}
	return broker
}

type Broker struct {
	conn *nats.Conn
}

func NewBroker(brokerURL string) (*Broker, error) {
	conn, err := nats.Connect(brokerURL)
	if err != nil {
		return nil, err
	}
	return &Broker{
		conn: conn,
	}, nil
}

func (b *Broker) PublishPingOneV2(ppm v3.ProposalPingMessage) error {
	bytes, err := json.Marshal(&ppm)
	if err != nil {
		return err
	}
	return b.conn.Publish("*.proposal-ping.v3", bytes)
}

func (b *Broker) PublishPingV2(ppm []v3.ProposalPingMessage) error {
	for _, p := range ppm {
		err := b.PublishPingOneV2(p)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *Broker) PublishUnregisterOneV2(pum v3.ProposalUnregisterMessage) error {
	bytes, err := json.Marshal(&pum)
	if err != nil {
		return err
	}
	return b.conn.Publish("*.proposal-unregister.v3", bytes)
}
