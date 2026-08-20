package price

import (
	"crypto/sha256"
	_ "embed"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed ui/earning-trends.html
var earningTrendsHTML []byte

//go:embed ui/earning-trends.css
var earningTrendsCSS []byte

//go:embed ui/earning-trends.js
var earningTrendsJS []byte

//go:embed ui/world.geojson
var worldGeoJSON []byte

var (
	earningTrendsCSSETag = contentETag(earningTrendsCSS)
	earningTrendsJSETag  = contentETag(earningTrendsJS)
	worldGeoJSONETag     = contentETag(worldGeoJSON)
)

func contentETag(content []byte) string {
	return fmt.Sprintf(`"%x"`, sha256.Sum256(content))
}

func (a *APIByService) EarningTrendsUI(c *gin.Context) {
	c.Header("Cache-Control", "no-cache")
	c.Header("Content-Security-Policy", "default-src 'self'; base-uri 'none'; frame-ancestors 'none'; object-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self' data:; font-src 'self'")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "text/html; charset=utf-8", earningTrendsHTML)
}

func (a *APIByService) EarningTrendsCSS(c *gin.Context) {
	serveEarningTrendsAsset(c, earningTrendsCSS, "text/css; charset=utf-8", earningTrendsCSSETag, "no-cache")
}

func (a *APIByService) EarningTrendsJS(c *gin.Context) {
	serveEarningTrendsAsset(c, earningTrendsJS, "application/javascript; charset=utf-8", earningTrendsJSETag, "no-cache")
}

func (a *APIByService) EarningTrendsMap(c *gin.Context) {
	serveEarningTrendsAsset(c, worldGeoJSON, "application/geo+json", worldGeoJSONETag, "public, max-age=86400")
}

func serveEarningTrendsAsset(c *gin.Context, content []byte, contentType, etag, cacheControl string) {
	c.Header("Cache-Control", cacheControl)
	c.Header("ETag", etag)
	c.Header("X-Content-Type-Options", "nosniff")
	if c.Request != nil && c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}
	c.Data(http.StatusOK, contentType, content)
}
