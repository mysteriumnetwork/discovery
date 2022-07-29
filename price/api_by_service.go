package price

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/discovery/price/pricingbyservice"
	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/rs/zerolog/log"
)

type APIByService struct {
	pricer    *pricingbyservice.PriceGetter
	jwtSecret string
	cfger     pricingbyservice.ConfigProvider
}

func NewAPIByService(pricer *pricingbyservice.PriceGetter, cfger pricingbyservice.ConfigProvider, jwtSecret string) *APIByService {
	return &APIByService{
		pricer:    pricer,
		cfger:     cfger,
		jwtSecret: jwtSecret,
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

func (a *APIByService) RegisterRoutes(r gin.IRoutes) {
	r.GET("/prices/config", JWTAuthorized(a.jwtSecret), a.GetConfig)
	r.POST("/prices/config", JWTAuthorized(a.jwtSecret), a.UpdateConfig)
	r.GET("/prices", a.LatestPrices)
}
