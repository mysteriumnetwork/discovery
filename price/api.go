package price

import (
	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/discovery/price/pricing"
)

type API struct {
	pricer *pricing.Pricer
}

func NewAPI(pricer *pricing.Pricer) *API {
	return &API{pricer: pricer}
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

func (a *API) RegisterRoutes(r gin.IRoutes) {
	r.GET("/prices", a.LatestPrices)
}
