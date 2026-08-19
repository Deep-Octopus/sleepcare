package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPathBodyLimitBoundsMatchingPublicRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(PathBodyLimit("/api/care/notification-provider-callbacks/", 17))
	engine.POST("/api/care/notification-provider-callbacks/:provider", func(c *gin.Context) {
		_, err := io.ReadAll(c.Request.Body)
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/api/care/notification-provider-callbacks/local", strings.NewReader(strings.Repeat("a", 64)))
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized callback status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
}
