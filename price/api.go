package price

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/discovery/gorest"
	"github.com/mysteriumnetwork/discovery/price/pricing"
	"github.com/rs/zerolog/log"
)

type API struct {
	pricer    *pricing.Pricer
	jwtSecret string
	cfger     pricing.ConfigProvider
}

func NewAPI(pricer *pricing.Pricer, cfger pricing.ConfigProvider, jwtSecret string) *API {
	return &API{
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
		c.JSON(http.StatusBadRequest, gorest.NewErrResponse(err.Error()))
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
		c.JSON(http.StatusBadRequest, gorest.NewErrResponse(err.Error()))
		return
	}

	err := a.cfger.Update(cfg)
	if err != nil {
		log.Err(err).Msg("Failed to update config")
		c.JSON(http.StatusBadRequest, gorest.NewErrResponse(err.Error()))
		return
	}

	c.Data(http.StatusAccepted, gin.MIMEJSON, nil)
}

func (a *API) RegisterRoutes(r gin.IRoutes) {
	r.GET("/prices/config", JWTAuthorized(a.jwtSecret), a.GetConfig)
	r.POST("/prices/config", JWTAuthorized(a.jwtSecret), a.UpdateConfig)
	r.GET("/prices", a.LatestPrices)
}

func JWTAuthorized(secret string) func(*gin.Context) {
	return func(c *gin.Context) {
		authHeader := strings.Split(c.Request.Header.Get("Authorization"), "Bearer ")
		if len(authHeader) != 2 {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				map[string]string{
					"error": "Malformed Token",
				},
			)
			return
		}
		jwtToken := authHeader[1]
		token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})
		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusUnauthorized,
				map[string]string{
					"error": "Unauthorized",
				},
			)
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			if !claims.VerifyExpiresAt(time.Now().Unix(), true) {
				c.AbortWithStatusJSON(
					http.StatusUnauthorized,
					map[string]string{
						"error": "Token expired",
					},
				)
				return
			}

			c.Next()
			return
		}

		c.AbortWithStatusJSON(
			http.StatusUnauthorized,
			map[string]string{
				"error": "Unauthorized",
			},
		)
	}
}
