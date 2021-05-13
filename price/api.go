package price

import (
	"math/big"
	"time"

	"github.com/gin-gonic/gin"
)

type API struct {
	latestPrices *LatestPrices
}

func NewAPI() *API {
	// TODO replace with organic pricing logic
	// current values are copy pasted from node
	pricePerGiBMax := new(big.Int).SetInt64(500_000_000_000_000_000) // 0.5 MYSTT
	pricePerHourMax := new(big.Int).SetInt64(180_000_000_000_000)    // 0.0018 MYSTT

	return &API{
		latestPrices: &LatestPrices{
			Current: &Prices{
				PricePerGiB:  new(big.Int).Div(pricePerGiBMax, big.NewInt(2)),
				PricePerHour: new(big.Int).Div(pricePerHourMax, big.NewInt(2)),
				ValidUntil:   time.Now().Add(time.Hour).UTC(),
			},
			Previous: &Prices{
				PricePerGiB:  new(big.Int).Div(pricePerGiBMax, big.NewInt(2)),
				PricePerHour: new(big.Int).Div(pricePerHourMax, big.NewInt(2)),
				ValidUntil:   time.Now().Add(time.Minute * 30 * -1).UTC(),
			},
		},
	}
}

func (a *API) LatestPrices(c *gin.Context) {
	c.JSON(200, a.latestPrices)
}

// LatestPrices holds two sets of prices. The Previous should be used in case
// a race condition between obtaining prices by Consumer and Provider
// upon agreement
type LatestPrices struct {
	Current  *Prices `json:"current"`
	Previous *Prices `json:"previous"`
}

type Prices struct {
	PricePerHour *big.Int  `json:"price_per_hour"`
	PricePerGiB  *big.Int  `json:"price_per_gib"`
	ValidUntil   time.Time `json:"valid_until"`
}

func (a *API) RegisterRoutes(r gin.IRoutes) {
	r.GET("/prices", a.LatestPrices)
}
