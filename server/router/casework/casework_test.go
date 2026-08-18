package casework

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCaseWorkRoutesMatchEmployeeContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	group := engine.Group("/api")
	(&CaseWorkRouter{}).InitCaseWorkRouter(group)

	want := map[string]bool{
		"GET /api/care/workbench":                             true,
		"GET /api/care/attention-cases":                       true,
		"GET /api/care/attention-cases/:id":                   true,
		"POST /api/care/attention-cases/:id/acknowledge":      true,
		"POST /api/care/attention-cases/:id/handling-records": true,
		"POST /api/care/attention-cases/:id/escalate":         true,
		"POST /api/care/attention-cases/:id/close":            true,
		"POST /api/care/attention-cases/:id/reopen":           true,
	}
	for _, route := range engine.Routes() {
		delete(want, route.Method+" "+route.Path)
	}
	if len(want) != 0 {
		t.Fatalf("missing case work routes: %+v", want)
	}
}
