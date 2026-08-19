package careclient

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCareClientRoutesMatchEmployeeContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	group := engine.Group("/api")
	(&CareClientRouter{}).InitCareClientRouter(group)

	want := map[string]bool{
		"GET /api/care/clients":                              true,
		"GET /api/care/clients/:id":                          true,
		"GET /api/care/client-options":                       true,
		"GET /api/care/data-governance-readiness":            true,
		"GET /api/care/clients/:id/data-lifecycle-requests":  true,
		"POST /api/care/clients":                             true,
		"PUT /api/care/clients/:id":                          true,
		"POST /api/care/clients/:id/assignments":             true,
		"POST /api/care/clients/:id/consent-records":         true,
		"POST /api/care/clients/:id/data-lifecycle-requests": true,
	}
	for _, route := range engine.Routes() {
		delete(want, route.Method+" "+route.Path)
	}
	if len(want) != 0 {
		t.Fatalf("missing care client routes: %+v", want)
	}
}
