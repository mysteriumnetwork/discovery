package middleware

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/mysteriumnetwork/token"
	"github.com/rs/zerolog/log"
	"io"
)

const discoveryAudienceName = "discovery"

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
		if err := j.newCheck(jwtToken); err != nil {
			log.Warn().Err(err).Msg("new jwt check failed")
			if err := j.oldCheck(jwtToken); err != nil {
				c.AbortWithStatusJSON(
					http.StatusUnauthorized,
					map[string]string{
						"error": err.Error(),
					},
				)
				return
			} else {
				log.Warn().Msg("old jwt token used")
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

type Token struct {
	Token string `json:"token"`
}

func (j *JWTChecker) valid(jwtToken string) (bool, error) {
	c := http.Client{Timeout: time.Second * 30}

	tokenBody, err := json.Marshal(
		Token{Token: jwtToken},
	)
	if err != nil {
		return false, err
	}
	resp, err := c.Post(fmt.Sprintf("%s/api/v1/token/validate", strings.TrimSuffix(j.SentinelURL, "/")), "application/json", body(string(tokenBody)))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode < 500 {
		return false, nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, nil
	}
	log.Error().Str("response", string(body)).Msg("failed to validate token: error response from validation server")
	return false, errors.New("failed to validate token: error response from validation server")
}

func (j *JWTChecker) newCheck(jtoken string) error {
	if valid, err := j.valid(jtoken); err != nil {
		return err
	} else if !valid {
		return errors.New("token invalid")
	}

	key, err := j.getPublicKey()
	if err != nil {
		return err
	}

	return token.NewValidatorJWT(key).ValidateForAudience(jtoken, discoveryAudienceName)
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

func body(in string) io.Reader {
	return strings.NewReader(in)
}
