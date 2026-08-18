package clientaccess

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/internal/testutil"
	clientmodel "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess"
	"github.com/gin-gonic/gin"
)

func TestRedeemRejectsMalformedJSONWithHTTP400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/redeem", new(ClientAccessApi).Redeem)
	request := httptest.NewRequest(http.MethodPost, "/redeem", strings.NewReader(`{"grant":`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), `"code":41001`) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestClientSessionCookieWindowUsesBusinessClockDuration(t *testing.T) {
	businessNow := time.Date(2026, time.August, 18, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	wallNow := time.Date(2026, time.August, 19, 10, 0, 0, 0, time.Local)

	maxAge, cookieExpiresAt := clientSessionCookieWindow(businessNow, businessNow.Add(8*time.Hour), wallNow)

	if maxAge != 8*60*60 {
		t.Fatalf("maxAge=%d want=%d", maxAge, 8*60*60)
	}
	if !cookieExpiresAt.Equal(wallNow.Add(8 * time.Hour)) {
		t.Fatalf("cookie expires at %s", cookieExpiresAt)
	}
}

func TestClientSessionCookieWindowKeepsExpiredSessionShortLived(t *testing.T) {
	now := time.Date(2026, time.August, 18, 10, 0, 0, 0, time.UTC)

	maxAge, cookieExpiresAt := clientSessionCookieWindow(now, now.Add(-time.Minute), now)

	if maxAge != 1 || !cookieExpiresAt.Equal(now.Add(time.Second)) {
		t.Fatalf("maxAge=%d expiresAt=%s", maxAge, cookieExpiresAt)
	}
}

func TestClientAccessErrorHTTPBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("domain scope denial", func(t *testing.T) {
		response := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(response)
		handleClientAccessError(ctx, clientmodel.NewHTTPError(clientmodel.CodeAccessScopeDenied, http.StatusForbidden, "任务不在访问范围"), "查询失败")
		if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), `"code":41104`) {
			t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
		}
	})

	t.Run("unexpected error", func(t *testing.T) {
		testutil.InitNopLogger(t)
		response := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(response)
		ctx.Request = httptest.NewRequest(http.MethodGet, "/client/tasks", nil)
		handleClientAccessError(ctx, errors.New("private database detail"), "查询失败")
		if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), "private database detail") {
			t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
		}
	})
}
