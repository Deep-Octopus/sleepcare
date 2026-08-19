package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// PathBodyLimit applies before AccessLog so a public JSON endpoint cannot make
// the global body capture allocate an unbounded request.
func PathBodyLimit(pathPrefix string, maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxBytes > 0 && strings.HasPrefix(c.Request.URL.Path, pathPrefix) {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
