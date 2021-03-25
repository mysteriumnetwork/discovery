package main

import (
	stdlog "log"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/discovery/db"
	"github.com/mysteriumnetwork/discovery/listener"
	"github.com/mysteriumnetwork/discovery/proposal"
	v2 "github.com/mysteriumnetwork/discovery/proposal/v2"
	"github.com/mysteriumnetwork/discovery/quality"
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
	qualityRepo := quality.NewRepository(rdb)
	brokerListener := listener.New("testnet2-broker.mysterium.network", proposalRepo)

	if err := brokerListener.Listen(); err != nil {
		log.Fatal().Err(err).Msg("Could not listen to the broker")
	}
	defer brokerListener.Shutdown()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.GET("/proposals", func(c *gin.Context) {
		list, err2 := proposalRepo.List("wireguard", "US")
		if err2 != nil {
			log.Err(err2).Msg("Failed to list proposals")
			c.JSON(500, "")
			return
		}

		qualities, err := qualityRepo.ListQualities(v2.ProposalProviderIDS(list), "wireguard", "DE")
		if err != nil {
			log.Err(err).Msg("failed listing proposal qualities")
			qualities = map[string]v2.Quality{}
		}

		for idx, p := range list {
			q, ok := qualities[p.ProviderID]
			if ok {
				list[idx].Quality = q
			}
		}

		c.JSON(200, list)
	})

	qa := quality.NewUpdater(
		"https://testnet2-quality.mysterium.network",
		qualityRepo,
		quality.UpdaterOpts{
			UpdateCycleDelay:  30 * time.Second,
			QualityFetchDelay: time.Second,
		},
	)
	go qa.Start()

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
