package clientaccess

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
