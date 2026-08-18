package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/gin-gonic/gin"
)

func TestClientSameOriginRequiresExactConfiguredOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previous := global.GVA_CONFIG.Care.ClientAccess.AllowedOrigins
	global.GVA_CONFIG.Care.ClientAccess.AllowedOrigins = []string{"http://127.0.0.1:8080"}
	t.Cleanup(func() {
		global.GVA_CONFIG.Care.ClientAccess.AllowedOrigins = previous
	})

	tests := []struct {
		name       string
		origin     string
		wantStatus int
		wantCalled bool
	}{
		{name: "exact", origin: "http://127.0.0.1:8080", wantStatus: http.StatusNoContent, wantCalled: true},
		{name: "missing", origin: "", wantStatus: http.StatusForbidden},
		{name: "different-port", origin: "http://127.0.0.1:8081", wantStatus: http.StatusForbidden},
		{name: "prefix-trick", origin: "http://127.0.0.1:8080.example.com", wantStatus: http.StatusForbidden},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			called := false
			router := gin.New()
			router.POST("/write", ClientSameOrigin(), func(c *gin.Context) {
				called = true
				c.Status(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodPost, "/write", nil)
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.wantStatus || called != test.wantCalled {
				t.Fatalf("status=%d called=%v, want status=%d called=%v", response.Code, called, test.wantStatus, test.wantCalled)
			}
		})
	}
}
