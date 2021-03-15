package main

import (
	"encoding/json"
	"errors"
	stdlog "log"
	"os"
	"reflect"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/gin-gonic/gin"
	v1 "github.com/mysteriumnetwork/discovery/proposal/v1"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var Version = "<dev>"

func main() {
	configureLogger()
	printBanner()
	r := gin.Default()

	broker, subscription, err := listenToBroker()
	if err != nil {
		log.Err(err).Msg("Could not listen to the broker")
		return
	}
	defer func() {
		subscription.Unsubscribe()
		broker.Close()
	}()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	if err := r.Run(); err != nil {
		log.Err(err).Send()
		return
	}

}

func listenToBroker() (*nats.Conn, *nats.Subscription, error) {
	nc, err := nats.Connect("testnet2-broker.mysterium.network")
	if err != nil {
		return nil, nil, err
	}
	log.Info().Msgf("Connected to broker: %v", nc.IsConnected())

	sub, err := nc.Subscribe("*.proposal-ping", func(msg *nats.Msg) {
		log.Info().Msgf("Received a message [%s] %s", msg.Subject, string(msg.Data))
		p := v1.ProposalPingMessage{}
		if err := json.Unmarshal(msg.Data, &p); err != nil {
			log.Err(err).Msg("Failed to parse proposal")
		} else if (reflect.DeepEqual(p, v1.ProposalPingMessage{})) {
			log.Err(errors.New("unknown message format")).Msg("Failed to parse proposal")
		} else {
			spew.Dump(p.Proposal)
		}
	})
	return nc, sub, err
}

func configureLogger() {
	writer := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02T15:04:05.000"}
	logger := log.Output(writer).Level(zerolog.DebugLevel).With().Caller().Timestamp().Logger()
	log.Logger = logger
	stdlog.SetFlags(0)
	stdlog.SetOutput(log.Logger)
}

func printBanner() {
	log.Info().Msg(strings.Repeat("▰", 60))
	log.Info().Msgf(" Starting discovery version: %s", Version)
	log.Info().Msg(strings.Repeat("▰", 60))
	log.Info().Msg(" She has carried us into the future")
	log.Info().Msg(" and it will be our privilege to make that future bright.")
	log.Info().Msg(strings.Repeat("▱", 60))
}
