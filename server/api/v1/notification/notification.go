package notification

import (
	"errors"
	"net/http"
	"strconv"

	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
	commonreq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	commonres "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	notificationmodel "github.com/flipped-aurora/gin-vue-admin/server/model/notification"
	notificationreq "github.com/flipped-aurora/gin-vue-admin/server/model/notification/request"
	notificationres "github.com/flipped-aurora/gin-vue-admin/server/model/notification/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/logger"
	"github.com/gin-gonic/gin"
)

type NotificationApi struct{}

var (
	_ notificationres.NotificationAttempt
	_ caseworkres.ActionResult
)

// ListDeliveries
// @Tags Deliveries
// @Summary 查询授权范围内通知尝试与标准回执
// @Security ApiKeyAuth
// @Produce application/json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量，最大100"
// @Param status query string false "通知尝试状态"
// @Success 200 {object} commonres.Response{data=commonres.PageResult{list=[]notificationres.NotificationAttempt},msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/deliveries [get]
func (a *NotificationApi) ListDeliveries(c *gin.Context) {
	var req notificationreq.DeliverySearch
	if err := c.ShouldBindQuery(&req); err != nil {
		commonres.FailWithMessage("查询参数无效", c)
		return
	}
	list, total, err := notificationService.ListDeliveries(c.Request.Context(), req)
	if err != nil {
		handleError(c, err, "查询通知记录失败")
		return
	}
	normalizePage(&req.PageInfo)
	commonres.OkWithDetailed(commonres.PageResult{
		List: list, Total: total, Page: req.Page, PageSize: req.PageSize,
	}, "查询成功", c)
}

// Resend
// @Tags Deliveries
// @Summary 为失败或未知终态创建新的通知尝试
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "原通知尝试ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body notificationreq.Resend true "原尝试版本和补发原因"
// @Success 200 {object} commonres.Response{data=caseworkres.ActionResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/deliveries/{id}/resend [post]
func (a *NotificationApi) Resend(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req notificationreq.Resend
	if err := c.ShouldBindJSON(&req); err != nil {
		commonres.FailWithMessage("请求参数无效", c)
		return
	}
	data, err := notificationService.Resend(c.Request.Context(), id, c.GetHeader("Idempotency-Key"), req)
	if err != nil {
		handleError(c, err, "创建补发尝试失败")
		return
	}
	commonres.OkWithDetailed(data, "补发尝试已创建", c)
}

func pathID(c *gin.Context) (uint, bool) {
	value, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || value == 0 {
		commonres.FailWithMessage("资源ID无效", c)
		return 0, false
	}
	return uint(value), true
}

func normalizePage(info *commonreq.PageInfo) {
	if info.Page <= 0 {
		info.Page = 1
	}
	if info.PageSize <= 0 {
		info.PageSize = 10
	}
	if info.PageSize > commonreq.MaxPageSize {
		info.PageSize = commonreq.MaxPageSize
	}
}

func handleError(c *gin.Context, err error, fallback string) {
	var domainErr *notificationmodel.DomainError
	if errors.As(err, &domainErr) {
		if domainErr.HTTPStatus == http.StatusForbidden {
			commonres.Forbidden(domainErr.Code, domainErr.Message, c)
			return
		}
		c.JSON(http.StatusOK, commonres.Response{Code: domainErr.Code, Data: nil, Msg: domainErr.Message})
		return
	}
	logger.WithCtx(c.Request.Context()).Mod("notification").Err(err).Error(fallback)
	commonres.FailWithMessage(fallback, c)
}
