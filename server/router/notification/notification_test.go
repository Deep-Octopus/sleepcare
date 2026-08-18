package notification

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNotificationRoutesMatchEmployeeContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	group := engine.Group("/api")
	(&NotificationRouter{}).InitNotificationRouter(group)

	want := map[string]bool{
		"GET /api/care/deliveries":             true,
		"POST /api/care/deliveries/:id/resend": true,
	}
	for _, route := range engine.Routes() {
		delete(want, route.Method+" "+route.Path)
	}
	for route := range want {
		t.Errorf("missing route %s", route)
	}
}
