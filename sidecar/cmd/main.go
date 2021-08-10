package main

import (
	"context"
	"fmt"
	stdlog "log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/mysteriumnetwork/discovery/config"
	"github.com/mysteriumnetwork/discovery/price/pricing"
	"github.com/mysteriumnetwork/discovery/quality/oracleapi"
	mlog "github.com/mysteriumnetwork/logger"
	payprice "github.com/mysteriumnetwork/payments/fees/price"
	"github.com/rs/zerolog/log"

	// unconfuse the number of cores go can use in k8s
	_ "go.uber.org/automaxprocs"
)

func main() {
	configureLogger()
	cfg, err := ReadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to read config")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddress,
		Password: cfg.RedisPass,
		DB:       cfg.RedisDB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	st := rdb.Ping(ctx)
	err = st.Err()
	if err != nil {
		log.Fatal().Err(err).Msg("could not reach redis")
	}
	cancel()

	qualityOracleAPI := oracleapi.New(cfg.QualityOracleURL.String())
	log.Info().Msg("qualityOracleAPI created")

	mrkt := buildMarket(cfg)
	err = mrkt.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("could not build market")
	}
	defer mrkt.Stop()
	log.Info().Msg("market started")

	cfger := pricing.NewConfigProviderDB(rdb)
	_, err = cfger.Get()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load cfg")
	}
	log.Info().Msg("cfger started")

	calc, err := buildLoadPricingProvider(cfg, qualityOracleAPI)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to build load pricing provider")
	}
	err = calc.Start()
	if err != nil {
		log.Fatal().Err(err).Msg("Could not start calculator")
	}
	log.Info().Msg("calculator started")
	defer calc.Stop()

	pricer, err := pricing.NewPricer(
		cfger,
		mrkt,
		time.Minute*5,
		pricing.Bound{Min: 0.1, Max: 3.0},
		calc,
		rdb,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Pricer")
		return
	}
	log.Info().Msg("pricer started")
	defer pricer.Stop()

	router := gin.New()
	router.Use(gin.Recovery())
	router.GET("/status", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := rdb.Ping(ctx).Err()
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		c.String(http.StatusOK, "OK")
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%v", getPort()),
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("listen: %s\n", err)
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	gctx, gcancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer gcancel()
	if err := srv.Shutdown(gctx); err != nil {
		log.Fatal().Err(err).Msg("server shutdown failed")
	}
}

func getPort() int {
	p := os.Getenv("PORT")
	if p == "" {
		return 8080
	}

	port, _ := strconv.Atoi(p)
	return port
}

func buildLoadPricingProvider(cfg *Options, oracle *oracleapi.API) (*pricing.NetworkLoadMultiplierCalculator, error) {
	calc := pricing.NewNetworkLoadMultiplierCalculator(
		oracle,
	)
	return calc, nil
}

func buildMarket(cfg *Options) *pricing.Market {
	apis := []pricing.ExternalPriceAPI{
		payprice.NewGecko(cfg.GeckoURL.String()),
		payprice.NewCoinRanking(cfg.CoinRankingURL.String(), &cfg.CoinRankingToken),
	}
	mrkt := pricing.NewMarket(apis, time.Minute*15)
	return mrkt
}

func configureLogger() {
	mlog.BootstrapDefaultLogger()
	stdlog.SetFlags(0)
	stdlog.SetOutput(log.Logger)
}

type Options struct {
	RedisAddress     string
	RedisPass        string
	RedisDB          int
	QualityOracleURL url.URL
	GeckoURL         url.URL
	CoinRankingURL   url.URL
	CoinRankingToken string
}

func ReadConfig() (*Options, error) {
	redisAddress, err := config.RequiredEnv("REDIS_ADDRESS")
	if err != nil {
		return nil, err
	}

	redisPass := config.OptionalEnv("REDIS_PASS", "")

	redisDBint := 0
	redisDB := config.OptionalEnv("REDIS_DB", "0")
	if redisDB != "" {
		res, err := strconv.Atoi(redisDB)
		if err != nil {
			return nil, fmt.Errorf("could not parse redis db from %q: %w", redisDB, err)
		}
		redisDBint = res
	}

	qualityOracleURL, err := config.RequiredEnvURL("QUALITY_ORACLE_URL")
	if err != nil {
		return nil, err
	}
	geckoURL, err := config.OptionalEnvURL("GECKO_URL", payprice.DefaultGeckoURI)
	if err != nil {
		return nil, err
	}
	coinRankingURL, err := config.OptionalEnvURL("COINRANKING_URL", payprice.DefaultCoinRankingURI)
	if err != nil {
		return nil, err
	}
	coinRankingToken, err := config.RequiredEnv("COINRANKING_TOKEN")
	if err != nil {
		return nil, err
	}
	return &Options{
		RedisAddress:     redisAddress,
		RedisPass:        redisPass,
		RedisDB:          redisDBint,
		QualityOracleURL: *qualityOracleURL,
		GeckoURL:         *geckoURL,
		CoinRankingURL:   *coinRankingURL,
		CoinRankingToken: coinRankingToken,
	}, nil
}
