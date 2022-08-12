package token

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/token"
)

type JWTChecker struct {
	SentinelURL string
	Secret      string

	publicKey []byte
}

func NewJWTChecker(sentinelURL, oldjwtSecret string) *JWTChecker {
	return &JWTChecker{
		SentinelURL: sentinelURL,
		Secret:      oldjwtSecret,
	}
}

// JWTAuthorized is a temporary hack to support BOTH new and old ways to authorize with a token.
// TODO: Eventually this should be fixed and only a single authorization method should be left
// which would be the sentinel auth. The struct could also then be removed and replaced with
// a simple middleware func.
func (j *JWTChecker) JWTAuthorized() func(*gin.Context) {
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
		if err := j.oldCheck(jwtToken); err != nil {
			if err := j.newCheck(jwtToken); err != nil {
				c.AbortWithStatusJSON(
					http.StatusUnauthorized,
					map[string]string{
						"error": err.Error(),
					},
				)
				return
			}
		}

		c.Next()
	}
}

type PublicKey struct {
	Key string `json:"key_base64"`
}

func (j *JWTChecker) getPublicKey() ([]byte, error) {
	if len(j.publicKey) > 0 {
		return j.publicKey, nil
	}
	c := http.Client{Timeout: time.Second * 30}
	resp, err := c.Get(fmt.Sprintf("%s/api/v1/auth/public/key", strings.TrimSuffix(j.SentinelURL, "/")))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %w", err)
	}

	var pb PublicKey
	if err = json.Unmarshal(body, &pb); err != nil {
		return nil, fmt.Errorf("could not unmarshal response body: %w", err)
	}

	got, err := base64.StdEncoding.DecodeString(pb.Key)
	if err != nil {
		return nil, err
	}
	j.publicKey = got

	return j.publicKey, nil
}

func (j *JWTChecker) newCheck(jtoken string) error {
	key, err := j.getPublicKey()
	if err != nil {
		return err
	}

	return token.NewValidatorJWT(key).Validate(jtoken)
}

func (j *JWTChecker) oldCheck(jtoken string) error {
	token, err := jwt.Parse(jtoken, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.Secret), nil
	})
	if err != nil {
		return errors.New("unauthorized")
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if !claims.VerifyExpiresAt(time.Now().Unix(), true) {
			return errors.New("expired")
		}
		return nil

	}

	return errors.New("token invalid")
}
