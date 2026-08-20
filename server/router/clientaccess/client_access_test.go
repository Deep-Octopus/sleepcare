package clientaccess

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestClientAccessRoutesMatchLimitedSessionContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	group := engine.Group("/api")
	(&ClientAccessRouter{}).InitClientAccessRouter(group)

	want := map[string]bool{
		"POST /api/care/client-auth/login":                          true,
		"POST /api/care/client-access/redeem":                       true,
		"GET /api/care/client/me":                                   true,
		"POST /api/care/client/logout":                              true,
		"GET /api/care/client/tasks":                                true,
		"GET /api/care/client/tasks/:taskId":                        true,
		"GET /api/care/client/tasks/:taskId/questionnaire":          true,
		"POST /api/care/client/tasks/:taskId/interactions":          true,
		"PUT /api/care/client/tasks/:taskId/draft":                  true,
		"POST /api/care/client/tasks/:taskId/submit":                true,
		"GET /api/care/client/consultations":                        true,
		"GET /api/care/client/consultations/:id":                    true,
		"POST /api/care/client/consultations":                       true,
		"POST /api/care/client/consultations/:id/messages":          true,
		"GET /api/care/client/satisfaction-requests":                true,
		"GET /api/care/client/satisfaction-requests/:id":            true,
		"POST /api/care/client/satisfaction-requests/:id/responses": true,
	}
	for _, route := range engine.Routes() {
		delete(want, route.Method+" "+route.Path)
	}
	if len(want) != 0 {
		t.Fatalf("missing limited-session routes: %+v", want)
	}
}
