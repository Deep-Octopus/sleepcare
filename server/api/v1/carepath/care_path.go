package carepath

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	pathmodel "github.com/flipped-aurora/gin-vue-admin/server/model/carepath"
	pathreq "github.com/flipped-aurora/gin-vue-admin/server/model/carepath/request"
	pathres "github.com/flipped-aurora/gin-vue-admin/server/model/carepath/response"
	commonres "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/logger"
	"github.com/gin-gonic/gin"
)

type CarePathApi struct{}

// ListPlanVersions
// @Tags CarePath
// @Summary 分页获取合成 OSA 计划模板版本
// @Security ApiKeyAuth
// @Produce application/json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量，最大100"
// @Param keyword query string false "编码、版本或标题"
// @Param status query string false "生命周期状态"
// @Param usageScope query string false "使用范围"
// @Param synthetic query bool false "是否合成版本"
// @Success 200 {object} commonres.Response{data=commonres.PageResult{list=[]pathres.PlanVersionSummary},msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/plan-versions [get]
func (a *CarePathApi) ListPlanVersions(c *gin.Context) {
	var req pathreq.PlanVersionSearch
	if err := c.ShouldBindQuery(&req); err != nil {
		commonres.FailWithMessage("查询参数无效", c)
		return
	}
	list, total, err := carePathService.ListPlanVersions(c.Request.Context(), req)
	if err != nil {
		handleCarePathError(c, err, "查询计划模板版本失败")
		return
	}
	page, pageSize := normalizedPage(req.Page, req.PageSize)
	commonres.OkWithDetailed(commonres.PageResult{List: list, Total: total, Page: page, PageSize: pageSize}, "查询成功", c)
}

// GetPlanVersion
// @Tags CarePath
// @Summary 获取合成 OSA 计划模板版本详情
// @Security ApiKeyAuth
// @Produce application/json
// @Param id path int true "计划模板版本ID"
// @Success 200 {object} commonres.Response{data=pathres.PlanVersionDetail,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/plan-versions/{id} [get]
func (a *CarePathApi) GetPlanVersion(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	data, err := carePathService.GetPlanVersion(c.Request.Context(), id)
	if err != nil {
		handleCarePathError(c, err, "查询计划模板版本详情失败")
		return
	}
	commonres.OkWithDetailed(data, "查询成功", c)
}

// PreviewPlan
// @Tags CarePath
// @Summary 按固定模板版本和 anchorAt 预览合成 D1-D5
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "康养用户ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body pathreq.PreviewPlan true "模板版本和合成锚点"
// @Success 200 {object} commonres.Response{data=pathres.PlanPreview,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/clients/{id}/plan-previews [post]
func (a *CarePathApi) PreviewPlan(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req pathreq.PreviewPlan
	if err := c.ShouldBindJSON(&req); err != nil {
		commonres.FailWithMessage("请求参数无效", c)
		return
	}
	data, err := carePathService.PreviewPlan(c.Request.Context(), id, c.GetHeader("Idempotency-Key"), req)
	if err != nil {
		handleCarePathError(c, err, "预览计划失败")
		return
	}
	commonres.OkWithDetailed(data, "预览成功", c)
}

// StartPlan
// @Tags CarePath
// @Summary 根据已确认预览幂等启动合成计划
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "康养用户ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body pathreq.StartPlan true "预览ID和康养用户乐观锁版本"
// @Success 200 {object} commonres.Response{data=pathres.PlanInstanceResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/clients/{id}/plan-instances [post]
func (a *CarePathApi) StartPlan(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req pathreq.StartPlan
	if err := c.ShouldBindJSON(&req); err != nil {
		commonres.FailWithMessage("请求参数无效", c)
		return
	}
	data, err := carePathService.StartPlan(c.Request.Context(), id, c.GetHeader("Idempotency-Key"), req)
	if err != nil {
		handleCarePathError(c, err, "启动计划失败")
		return
	}
	commonres.OkWithDetailed(data, "计划已启动", c)
}

// ListClientPlans
// @Tags CarePath
// @Summary 获取康养用户责任范围内的计划时间线
// @Security ApiKeyAuth
// @Produce application/json
// @Param id path int true "康养用户ID"
// @Success 200 {object} commonres.Response{data=[]pathres.PlanInstanceSummary,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/clients/{id}/plan-instances [get]
func (a *CarePathApi) ListClientPlans(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	data, err := carePathService.ListClientPlans(c.Request.Context(), id)
	if err != nil {
		handleCarePathError(c, err, "查询用户计划失败")
		return
	}
	commonres.OkWithDetailed(data, "查询成功", c)
}

// PausePlan
// @Tags CarePath
// @Summary 追加原因并按 KEEP_WINDOWS 暂停合成计划
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "计划实例ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body pathreq.PlanStateAction true "乐观锁版本和原因"
// @Success 200 {object} commonres.Response{data=pathres.PlanActionResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/plan-instances/{id}/pause [post]
func (a *CarePathApi) PausePlan(c *gin.Context) {
	a.changePlanState(c, carePathService.PausePlan, "计划已暂停")
}

// ResumePlan
// @Tags CarePath
// @Summary 追加原因并按 KEEP_WINDOWS 恢复合成计划
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "计划实例ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body pathreq.PlanStateAction true "乐观锁版本和原因"
// @Success 200 {object} commonres.Response{data=pathres.PlanActionResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/plan-instances/{id}/resume [post]
func (a *CarePathApi) ResumePlan(c *gin.Context) {
	a.changePlanState(c, carePathService.ResumePlan, "计划已恢复")
}

func (a *CarePathApi) changePlanState(c *gin.Context, action func(context.Context, uint, string, pathreq.PlanStateAction) (pathres.PlanActionResult, error), message string) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req pathreq.PlanStateAction
	if err := c.ShouldBindJSON(&req); err != nil {
		commonres.FailWithMessage("请求参数无效", c)
		return
	}
	data, err := action(c.Request.Context(), id, c.GetHeader("Idempotency-Key"), req)
	if err != nil {
		handleCarePathError(c, err, "更新计划状态失败")
		return
	}
	commonres.OkWithDetailed(data, message, c)
}

// ListTasks
// @Tags CarePath
// @Summary 分页获取员工责任范围内的计划任务
// @Security ApiKeyAuth
// @Produce application/json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量，最大100"
// @Param careClientId query int false "康养用户ID"
// @Param planInstanceId query int false "计划实例ID"
// @Param executionStatus query string false "执行状态"
// @Param timingStatus query string false "时效状态"
// @Param dayCode query string false "D1-D5"
// @Success 200 {object} commonres.Response{data=commonres.PageResult{list=[]pathres.TaskSummary},msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/tasks [get]
func (a *CarePathApi) ListTasks(c *gin.Context) {
	var req pathreq.TaskSearch
	if err := c.ShouldBindQuery(&req); err != nil {
		commonres.FailWithMessage("查询参数无效", c)
		return
	}
	list, total, err := carePathService.ListTasks(c.Request.Context(), req)
	if err != nil {
		handleCarePathError(c, err, "查询计划任务失败")
		return
	}
	page, pageSize := normalizedPage(req.Page, req.PageSize)
	commonres.OkWithDetailed(commonres.PageResult{List: list, Total: total, Page: page, PageSize: pageSize}, "查询成功", c)
}

// GetTask
// @Tags CarePath
// @Summary 获取员工责任范围内的计划任务详情
// @Security ApiKeyAuth
// @Produce application/json
// @Param id path int true "任务实例ID"
// @Success 200 {object} commonres.Response{data=pathres.TaskDetail,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/tasks/{id} [get]
func (a *CarePathApi) GetTask(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	data, err := carePathService.GetTask(c.Request.Context(), id)
	if err != nil {
		handleCarePathError(c, err, "查询计划任务详情失败")
		return
	}
	commonres.OkWithDetailed(data, "查询成功", c)
}

// RecordTaskContact
// @Tags CarePath
// @Summary 为责任范围内任务追加人工联系记录
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "任务实例ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body pathreq.TaskContactRecord true "任务版本、联系渠道、结果和发生时间"
// @Success 200 {object} commonres.Response{data=pathres.TaskActionResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/tasks/{id}/contact-records [post]
func (a *CarePathApi) RecordTaskContact(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req pathreq.TaskContactRecord
	if err := c.ShouldBindJSON(&req); err != nil {
		commonres.FailWithMessage("请求参数无效", c)
		return
	}
	data, err := carePathService.RecordTaskContact(c.Request.Context(), id, c.GetHeader("Idempotency-Key"), req)
	if err != nil {
		handleCarePathError(c, err, "记录人工联系失败")
		return
	}
	commonres.OkWithDetailed(data, "联系记录已追加", c)
}

func pathID(c *gin.Context) (uint, bool) {
	value, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || value == 0 {
		commonres.FailWithMessage("资源ID无效", c)
		return 0, false
	}
	return uint(value), true
}

func normalizedPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func handleCarePathError(c *gin.Context, err error, fallback string) {
	var domainErr *pathmodel.DomainError
	if errors.As(err, &domainErr) {
		if domainErr.HTTPStatus == http.StatusForbidden {
			commonres.Forbidden(domainErr.Code, domainErr.Message, c)
			return
		}
		c.JSON(http.StatusOK, commonres.Response{Code: domainErr.Code, Data: nil, Msg: domainErr.Message})
		return
	}
	logger.WithCtx(c.Request.Context()).Mod("carepath").Err(err).Error(fallback)
	commonres.FailWithMessage(fallback, c)
}
