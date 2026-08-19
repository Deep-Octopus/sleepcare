package aiassist

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAIShadowRoutesExposeOnlyReadiness(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	private := engine.Group("/api")
	(&AIShadowRouter{}).InitAIShadowRouter(private)

	routes := engine.Routes()
	if len(routes) != 1 {
		t.Fatalf("AI shadow route count = %d, want 1", len(routes))
	}
	if routes[0].Method != "GET" || routes[0].Path != "/api/care/ai-shadow-readiness" {
		t.Fatalf("unexpected AI shadow route: %s %s", routes[0].Method, routes[0].Path)
	}
}
