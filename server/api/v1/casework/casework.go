package casework

import (
	"errors"
	"net/http"
	"strconv"

	caseworkmodel "github.com/flipped-aurora/gin-vue-admin/server/model/casework"
	caseworkreq "github.com/flipped-aurora/gin-vue-admin/server/model/casework/request"
	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
	commonreq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	commonres "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/logger"
	"github.com/gin-gonic/gin"
)

type CaseWorkApi struct{}

// Keep concrete response packages visible to both the compiler and swag.
var (
	_ caseworkres.ActionResult
	_ caseworkres.AttentionCaseDetail
)

// ListAttentionCases
// @Tags AttentionCase
// @Summary 分页获取授权范围内的关注事项
// @Security ApiKeyAuth
// @Produce application/json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量，最大100"
// @Param status query string false "事项状态"
// @Param assigneeId query int false "责任人ID，只能缩小授权范围"
// @Success 200 {object} commonres.Response{data=commonres.PageResult{list=[]caseworkres.AttentionCaseSummary},msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/attention-cases [get]
func (a *CaseWorkApi) ListAttentionCases(c *gin.Context) {
	var req caseworkreq.AttentionCaseSearch
	if err := c.ShouldBindQuery(&req); err != nil {
		commonres.FailWithMessage("查询参数无效", c)
		return
	}
	list, total, err := caseWorkService.List(c.Request.Context(), req)
	if err != nil {
		handleError(c, err, "查询关注事项失败")
		return
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > commonreq.MaxPageSize {
		req.PageSize = commonreq.MaxPageSize
	}
	commonres.OkWithDetailed(commonres.PageResult{
		List: list, Total: total, Page: req.Page, PageSize: req.PageSize,
	}, "查询成功", c)
}

// GetAttentionCase
// @Tags AttentionCase
// @Summary 获取授权范围内的关注事项详情
// @Security ApiKeyAuth
// @Produce application/json
// @Param id path int true "关注事项ID"
// @Success 200 {object} commonres.Response{data=caseworkres.AttentionCaseDetail,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/attention-cases/{id} [get]
func (a *CaseWorkApi) GetAttentionCase(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	data, err := caseWorkService.Get(c.Request.Context(), id)
	if err != nil {
		handleError(c, err, "查询关注事项详情失败")
		return
	}
	commonres.OkWithDetailed(data, "查询成功", c)
}

// AcknowledgeAttentionCase
// @Tags AttentionCase
// @Summary 确认关注事项
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "关注事项ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body caseworkreq.AcknowledgeCase true "确认结果和乐观锁版本"
// @Success 200 {object} commonres.Response{data=caseworkres.ActionResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/attention-cases/{id}/acknowledge [post]
func (a *CaseWorkApi) AcknowledgeAttentionCase(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req caseworkreq.AcknowledgeCase
	if err := c.ShouldBindJSON(&req); err != nil {
		commonres.FailWithMessage("请求参数无效", c)
		return
	}
	data, err := caseWorkService.Acknowledge(c.Request.Context(), id, c.GetHeader("Idempotency-Key"), req)
	if err != nil {
		handleError(c, err, "确认关注事项失败")
		return
	}
	commonres.OkWithDetailed(data, "确认成功", c)
}

// CreateHandlingRecord
// @Tags AttentionCase
// @Summary 追加联系或处置记录
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "关注事项ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body caseworkreq.HandlingRecord true "动作、结果和乐观锁版本"
// @Success 200 {object} commonres.Response{data=caseworkres.ActionResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/attention-cases/{id}/handling-records [post]
func (a *CaseWorkApi) CreateHandlingRecord(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req caseworkreq.HandlingRecord
	if err := c.ShouldBindJSON(&req); err != nil {
		commonres.FailWithMessage("请求参数无效", c)
		return
	}
	data, err := caseWorkService.RecordHandling(c.Request.Context(), id, c.GetHeader("Idempotency-Key"), req)
	if err != nil {
		handleError(c, err, "记录关注事项处理失败")
		return
	}
	commonres.OkWithDetailed(data, "记录成功", c)
}

// EscalateAttentionCase
// @Tags AttentionCase
// @Summary 升级关注事项并保留责任链
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "关注事项ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body caseworkreq.EscalateCase true "升级目标、理由和乐观锁版本"
// @Success 200 {object} commonres.Response{data=caseworkres.ActionResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/attention-cases/{id}/escalate [post]
func (a *CaseWorkApi) EscalateAttentionCase(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req caseworkreq.EscalateCase
	if err := c.ShouldBindJSON(&req); err != nil {
		commonres.FailWithMessage("请求参数无效", c)
		return
	}
	data, err := caseWorkService.Escalate(c.Request.Context(), id, c.GetHeader("Idempotency-Key"), req)
	if err != nil {
		handleError(c, err, "升级关注事项失败")
		return
	}
	commonres.OkWithDetailed(data, "升级成功", c)
}

// CloseAttentionCase
// @Tags AttentionCase
// @Summary 在处理结果和关闭理由完整后关闭事项
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "关注事项ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body caseworkreq.CloseCase true "关闭理由和乐观锁版本"
// @Success 200 {object} commonres.Response{data=caseworkres.ActionResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/attention-cases/{id}/close [post]
func (a *CaseWorkApi) CloseAttentionCase(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req caseworkreq.CloseCase
	if err := c.ShouldBindJSON(&req); err != nil {
		commonres.FailWithMessage("请求参数无效", c)
		return
	}
	data, err := caseWorkService.Close(c.Request.Context(), id, c.GetHeader("Idempotency-Key"), req)
	if err != nil {
		handleError(c, err, "关闭关注事项失败")
		return
	}
	commonres.OkWithDetailed(data, "关闭成功", c)
}

// ReopenAttentionCase
// @Tags AttentionCase
// @Summary 由主管重开已关闭事项并保留历史
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "关注事项ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body caseworkreq.ReopenCase true "重开理由和乐观锁版本"
// @Success 200 {object} commonres.Response{data=caseworkres.ActionResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/attention-cases/{id}/reopen [post]
func (a *CaseWorkApi) ReopenAttentionCase(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req caseworkreq.ReopenCase
	if err := c.ShouldBindJSON(&req); err != nil {
		commonres.FailWithMessage("请求参数无效", c)
		return
	}
	data, err := caseWorkService.Reopen(c.Request.Context(), id, c.GetHeader("Idempotency-Key"), req)
	if err != nil {
		handleError(c, err, "重开关注事项失败")
		return
	}
	commonres.OkWithDetailed(data, "重开成功", c)
}

func pathID(c *gin.Context) (uint, bool) {
	value, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || value == 0 {
		commonres.FailWithMessage("资源ID无效", c)
		return 0, false
	}
	return uint(value), true
}

func handleError(c *gin.Context, err error, fallback string) {
	var domainErr *caseworkmodel.DomainError
	if errors.As(err, &domainErr) {
		if domainErr.HTTPStatus == http.StatusForbidden {
			commonres.Forbidden(domainErr.Code, domainErr.Message, c)
			return
		}
		c.JSON(http.StatusOK, commonres.Response{Code: domainErr.Code, Data: nil, Msg: domainErr.Message})
		return
	}
	logger.WithCtx(c.Request.Context()).Mod("casework").Err(err).Error(fallback)
	commonres.FailWithMessage(fallback, c)
}
