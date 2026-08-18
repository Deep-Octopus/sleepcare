package carepath

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCarePathRoutesMatchEmployeeContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	group := engine.Group("/api")
	(&CarePathRouter{}).InitCarePathRouter(group)

	want := map[string]bool{
		"POST /api/care/clients/:id/plan-previews":  true,
		"POST /api/care/clients/:id/plan-instances": true,
		"POST /api/care/plan-instances/:id/pause":   true,
		"POST /api/care/plan-instances/:id/resume":  true,
		"POST /api/care/tasks/:id/contact-records":  true,
		"GET /api/care/plan-versions":               true,
		"GET /api/care/plan-versions/:id":           true,
		"GET /api/care/clients/:id/plan-instances":  true,
		"GET /api/care/tasks":                       true,
		"GET /api/care/tasks/:id":                   true,
	}
	for _, route := range engine.Routes() {
		delete(want, route.Method+" "+route.Path)
	}
	for route := range want {
		t.Errorf("missing route %s", route)
	}
}
