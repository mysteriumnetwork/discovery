package price

import (
	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/discovery/price/pricing"
)

type API struct {
}

func NewAPI() *API {
	return &API{}
}

func (a *API) LatestPrices(c *gin.Context) {
	c.JSON(200, pricing.LatestPrices{})
}

func (a *API) RegisterRoutes(r gin.IRoutes) {
	r.GET("/prices", a.LatestPrices)
}
