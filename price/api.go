package price

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/discovery/price/pricing"
	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/rs/zerolog/log"
)

const (
	errCodeParsingJson = "err_parsing_config"

	errCodeNoConfig     = "err_no_config"
	errCodeUpdateConfig = "err_update_config"
)

type API struct {
	pricer *pricing.PriceGetter
	cfger  pricing.ConfigProvider

	ac authCheck
}

func NewAPI(pricer *pricing.PriceGetter, cfger pricing.ConfigProvider, ac authCheck) *API {
	return &API{
		pricer: pricer,
		cfger:  cfger,
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
func (a *API) LatestPrices(c *gin.Context) {
	c.JSON(200, a.pricer.GetPrices())
}

// GetConfig returns the base pricing config
// @Summary Price config
// @Description price config
// @Product json
// @Success 200 {array} pricing.Config
// @Router /prices/config [get]
// @Tags prices
func (a *API) GetConfig(c *gin.Context) {
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
func (a *API) UpdateConfig(c *gin.Context) {
	var cfg pricing.Config
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

func (a *API) RegisterRoutes(r gin.IRoutes) {
	r.GET("/prices/config", a.ac.JWTAuthorized(), a.GetConfig)
	r.POST("/prices/config", a.ac.JWTAuthorized(), a.UpdateConfig)
	r.GET("/prices", a.LatestPrices)
}
