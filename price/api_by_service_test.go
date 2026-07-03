package price

import (
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/mysteriumnetwork/discovery/middleware"
	"github.com/mysteriumnetwork/discovery/price/pricingbyservice"
)

type staticLatestPricer struct {
	prices pricingbyservice.LatestPrices
}

func (s staticLatestPricer) GetPrices() pricingbyservice.LatestPrices {
	return s.prices
}

func TestLatestPricesReturnsErrorWhenJSONMarshalFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	api := &APIByService{
		pricer: staticLatestPricer{
			prices: pricingbyservice.LatestPrices{
				Defaults: &pricingbyservice.PriceHistory{
					Current: &pricingbyservice.PriceByType{
						Residential: &pricingbyservice.PriceByServiceType{
							Wireguard: pricingbyservice.Price{
								PricePerHourHumanReadable: math.NaN(),
							},
						},
					},
				},
			},
		},
	}

	router := gin.New()
	router.Use(middleware.ErrorHandler)
	router.GET("/api/v4/prices", api.LatestPrices)

	req := httptest.NewRequest(http.MethodGet, "/api/v4/prices", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(resp.Body.String(), errCodeMarshalJson) {
		t.Fatalf("response body = %q, want error code %q", resp.Body.String(), errCodeMarshalJson)
	}
}
