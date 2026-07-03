package middleware

import (
	"encoding/json"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/rs/zerolog/log"
)

// ErrorHandler formats request errors unless the response has already been sent.
func ErrorHandler(c *gin.Context) {
	c.Next()
	if len(c.Errors) < 1 {
		return
	}
	if c.Writer.Written() {
		log.Err(c.Errors[0].Err).Msg("response already written, skipping error response")
		return
	}

	err := c.Errors[0].Err
	var apiErr *apierror.APIError
	if !errors.As(err, &apiErr) {
		apiErr = apierror.Internal(err.Error(), apierror.ErrCodeInternal)
	}
	apiErr.Path = c.Request.URL.String()

	blob, err := json.Marshal(apiErr)
	if err != nil {
		c.Data(500, apierror.ContentTypeV1, apierror.DefaultErrStatic)
		return
	}
	c.Data(apiErr.Status, apierror.ContentTypeV1, blob)
}
