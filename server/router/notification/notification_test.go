package notification

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNotificationRoutesMatchEmployeeContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	private := engine.Group("/api")
	public := engine.Group("/api")
	(&NotificationRouter{}).InitNotificationRouter(private, public)

	want := map[string]bool{
		"GET /api/care/deliveries":                                     true,
		"GET /api/care/notification-provider-readiness":                true,
		"POST /api/care/deliveries/:id/resend":                         true,
		"POST /api/care/notification-provider-callbacks/:providerCode": true,
	}
	for _, route := range engine.Routes() {
		delete(want, route.Method+" "+route.Path)
	}
	for route := range want {
		t.Errorf("missing route %s", route)
	}
}
