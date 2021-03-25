package main

import (
	stdlog "log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/discovery/db"
	"github.com/mysteriumnetwork/discovery/listener"
	"github.com/mysteriumnetwork/discovery/proposal"
	"github.com/mysteriumnetwork/discovery/quality"
	"github.com/mysteriumnetwork/discovery/quality/oracleapi"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var Version = "<dev>"

func main() {
	configureLogger()
	printBanner()
	r := gin.Default()

	rdb := db.New()
	proposalRepo := proposal.NewRepository(rdb)
	qualityOracleAPI := oracleapi.New("https://testnet2-quality.mysterium.network")
	qualityService := quality.NewService(qualityOracleAPI, rdb)
	proposalService := proposal.NewService(proposalRepo, qualityService)

	proposal.NewAPI(proposalService).RegisterRoutes(r)

	brokerListener := listener.New("testnet2-broker.mysterium.network", proposalRepo)

	if err := brokerListener.Listen(); err != nil {
		log.Fatal().Err(err).Msg("Could not listen to the broker")
	}
	defer brokerListener.Shutdown()

	if err := r.Run(); err != nil {
		log.Err(err).Send()
		return
	}
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
