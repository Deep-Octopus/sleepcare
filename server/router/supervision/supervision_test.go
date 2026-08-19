package supervision

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSupervisionRoutesMatchEmployeeContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	group := engine.Group("/api")
	(&SupervisionRouter{}).InitSupervisionRouter(group)

	want := map[string]bool{
		"GET /api/care/daily-summaries":                          true,
		"GET /api/care/daily-summaries/:id":                      true,
		"GET /api/care/reviews":                                  true,
		"POST /api/care/reviews/:id/guidance":                    true,
		"POST /api/care/reviews/:id/intervene":                   true,
		"GET /api/care/satisfaction-responses":                   true,
		"GET /api/care/satisfaction-follow-ups":                  true,
		"GET /api/care/satisfaction-follow-ups/:id":              true,
		"POST /api/care/satisfaction-follow-ups/:id/acknowledge": true,
		"POST /api/care/satisfaction-follow-ups/:id/resolve":     true,
	}
	for _, route := range engine.Routes() {
		delete(want, route.Method+" "+route.Path)
	}
	for route := range want {
		t.Errorf("missing route %s", route)
	}
}
