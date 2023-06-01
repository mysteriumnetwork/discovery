package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func Logger(c *gin.Context) {
	t := time.Now().UTC()
	// before request
	path := c.Request.URL.Path
	raw := c.Request.URL.RawQuery
	if raw != "" {
		path = path + "?" + raw
	}
	c.Next()

	statusCode := c.Writer.Status()
	f := map[string]any{
		"status_code": statusCode,
		"method":      c.Request.Method,
		"path":        path,
		"latency":     time.Since(t).Seconds(),
		"clientIP":    c.ClientIP(),
	}

	msg := "API Request"
	switch {
	case statusCode >= 400 && statusCode < 500:
		log.Debug().Fields(f).Msg(msg)
	case statusCode >= 500:
		log.Error().Fields(f).Msg(msg)
	default:
		log.Debug().Fields(f).Msg(msg)
	}
}
