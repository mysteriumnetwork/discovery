package price

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/discovery/price/pricingbyservice"
	"github.com/mysteriumnetwork/go-rest/apierror"
)

type APIByService struct {
	pricer *pricingbyservice.PriceGetter
	cfger  pricingbyservice.ConfigProvider
	redis  redis.UniversalClient

	ac authCheck
}

type authCheck interface {
	JWTAuthorized() func(*gin.Context)
}

func NewAPIByService(redis redis.UniversalClient, pricer *pricingbyservice.PriceGetter, cfger pricingbyservice.ConfigProvider, ac authCheck) *APIByService {
	return &APIByService{
		pricer: pricer,
		cfger:  cfger,
		redis:  redis,
		ac:     ac,
	}
}

// LatestPrices returns latest prices
// @Summary Latest Prices
// @Description Latest Prices
// @Product json
// @Success 200 {array} pricing.LatestPrices
// @Router /prices [get]
// @Tags prices
func (a *APIByService) LatestPrices(c *gin.Context) {
	c.JSON(200, a.pricer.GetPrices())
}

// GetConfig returns the base pricing config
// @Summary Price config
// @Description price config
// @Product json
// @Success 200 {array} pricing.Config
// @Router /prices/config [get]
// @Tags prices
func (a *APIByService) GetConfig(c *gin.Context) {
	cfg, err := a.cfger.Get()
	if err != nil {
		log.Err(err).Msg("Failed to get config")
		c.Error(apierror.Internal(err.Error(), errCodeNoConfig))
		return
	}
	c.JSON(http.StatusOK, cfg)
}

// UpdateConfig updates the pricing config
// @Summary update price config
// @Description update price config
// @Product json
// @Success 202
// @Param config body pricing.Config true "config object"
// @Router /prices/config [post]
// @Tags prices
func (a *APIByService) UpdateConfig(c *gin.Context) {
	var cfg pricingbyservice.Config
	if err := c.BindJSON(&cfg); err != nil {
		c.Error(apierror.BadRequest(err.Error(), errCodeParsingJson))
		return
	}

	err := a.cfger.Update(cfg)
	if err != nil {
		log.Err(err).Msg("Failed to update config")
		c.Error(apierror.BadRequest(err.Error(), errCodeUpdateConfig))
		return
	}

	c.Data(http.StatusAccepted, gin.MIMEJSON, nil)
}

// Status godoc.
// @Summary Status
// @Description Status
// @Accept json
// @Produce json
// @Success 200 {object} StatusResponse
// @Router /status [get]
// @Tags system
func (a *APIByService) Status(c *gin.Context) {
	sr := StatusResponse{
		CacheOK: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := a.redis.Ping(ctx).Err()
	if err != nil {
		sr.CacheOK = false
		log.Err(err).Msg("could not reach redis")
		c.Error(apierror.Internal(err.Error(), errRedisPingFailed))
		return
	}

	c.JSON(200, sr)
}

type StatusResponse struct {
	CacheOK bool `json:"cache_ok"`
}

// Ping godoc.
// @Summary Ping
// @Description Ping
// @Accept json
// @Produce json
// @Success 200 {object} PingResponse
// @Router /ping [get]
// @Tags system
func (a *APIByService) Ping(c *gin.Context) {
	c.JSON(200, PingResponse{"pong"})
}

type PingResponse struct {
	Message string `json:"message"`
}

func (a *APIByService) RegisterRoutes(r gin.IRoutes) {
	r.GET("/prices/config", a.ac.JWTAuthorized(), a.GetConfig)
	r.POST("/prices/config", a.ac.JWTAuthorized(), a.UpdateConfig)
	r.GET("/prices", a.LatestPrices)
	r.GET("/ping", a.Ping)
	r.GET("/status", a.Status)
}
