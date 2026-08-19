package casework

import (
	caseworkreq "github.com/flipped-aurora/gin-vue-admin/server/model/casework/request"
	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
	commonreq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	commonres "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
)

var (
	_ caseworkres.ConsultationActionResult
	_ caseworkres.ConsultationAssigneeOption
	_ caseworkres.ConsultationDetail
)

// ListConsultations
// @Tags Consultation
// @Summary 分页获取授权责任范围内的主动咨询
// @Security ApiKeyAuth
// @Produce application/json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量，最大100"
// @Param status query string false "咨询状态"
// @Param urgency query string false "联系优先级"
// @Param assigneeId query int false "责任人ID，只能缩小授权范围"
// @Success 200 {object} commonres.Response{data=commonres.PageResult{list=[]caseworkres.ConsultationSummary},msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/consultations [get]
func (a *CaseWorkApi) ListConsultations(c *gin.Context) {
	var req caseworkreq.ConsultationSearch
	if err := c.ShouldBindQuery(&req); err != nil {
		commonres.FailWithMessage("查询参数无效", c)
		return
	}
	list, total, err := caseWorkService.ListConsultations(c.Request.Context(), req)
	if err != nil {
		handleError(c, err, "查询咨询失败")
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
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, "查询成功", c)
}

// GetConsultation
// @Tags Consultation
// @Summary 获取授权责任范围内的主动咨询详情
// @Security ApiKeyAuth
// @Produce application/json
// @Param id path int true "咨询ID"
// @Success 200 {object} commonres.Response{data=caseworkres.ConsultationDetail,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/consultations/{id} [get]
func (a *CaseWorkApi) GetConsultation(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	data, err := caseWorkService.GetConsultation(c.Request.Context(), id)
	if err != nil {
		handleError(c, err, "查询咨询详情失败")
		return
	}
	commonres.OkWithDetailed(data, "查询成功", c)
}

// ListConsultationAssigneeOptions
// @Tags Consultation
// @Summary 获取当前咨询可用的责任人员
// @Security ApiKeyAuth
// @Produce application/json
// @Param id path int true "咨询ID"
// @Success 200 {object} commonres.Response{data=[]caseworkres.ConsultationAssigneeOption,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/consultations/{id}/assignee-options [get]
func (a *CaseWorkApi) ListConsultationAssigneeOptions(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	data, err := caseWorkService.ListConsultationAssigneeOptions(c.Request.Context(), id)
	if err != nil {
		handleError(c, err, "查询咨询责任人员失败")
		return
	}
	commonres.OkWithDetailed(data, "查询成功", c)
}

// AssignConsultation
// @Tags Consultation
// @Summary 上级为待分配咨询指定责任人员
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "咨询ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body caseworkreq.AssignConsultation true "版本、目标责任人和原因"
// @Success 200 {object} commonres.Response{data=caseworkres.ConsultationActionResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/consultations/{id}/assign [post]
func (a *CaseWorkApi) AssignConsultation(c *gin.Context) {
	a.handleConsultationCommand(c, new(caseworkreq.AssignConsultation), func(id uint, value any) (caseworkres.ConsultationActionResult, error) {
		return caseWorkService.AssignConsultation(c.Request.Context(), id, c.GetHeader("Idempotency-Key"), *value.(*caseworkreq.AssignConsultation))
	}, "分配咨询失败", "咨询已分配")
}

// ReplyConsultation
// @Tags Consultation
// @Summary 当前责任人公开回复咨询
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "咨询ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body caseworkreq.ReplyConsultation true "版本、回复和后续状态"
// @Success 200 {object} commonres.Response{data=caseworkres.ConsultationActionResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/consultations/{id}/replies [post]
func (a *CaseWorkApi) ReplyConsultation(c *gin.Context) {
	a.handleConsultationCommand(c, new(caseworkreq.ReplyConsultation), func(id uint, value any) (caseworkres.ConsultationActionResult, error) {
		return caseWorkService.ReplyConsultation(c.Request.Context(), id, c.GetHeader("Idempotency-Key"), *value.(*caseworkreq.ReplyConsultation))
	}, "回复咨询失败", "回复已记录")
}

// TransferConsultation
// @Tags Consultation
// @Summary 当前责任人转交咨询
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "咨询ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body caseworkreq.TransferConsultation true "版本、目标责任人和原因"
// @Success 200 {object} commonres.Response{data=caseworkres.ConsultationActionResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/consultations/{id}/transfer [post]
func (a *CaseWorkApi) TransferConsultation(c *gin.Context) {
	a.handleConsultationCommand(c, new(caseworkreq.TransferConsultation), func(id uint, value any) (caseworkres.ConsultationActionResult, error) {
		return caseWorkService.TransferConsultation(c.Request.Context(), id, c.GetHeader("Idempotency-Key"), *value.(*caseworkreq.TransferConsultation))
	}, "转交咨询失败", "咨询已转交")
}

// EscalateConsultation
// @Tags Consultation
// @Summary 当前责任人升级咨询并保留责任链
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "咨询ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body caseworkreq.EscalateConsultation true "版本、目标人员和升级原因"
// @Success 200 {object} commonres.Response{data=caseworkres.ConsultationActionResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/consultations/{id}/escalate [post]
func (a *CaseWorkApi) EscalateConsultation(c *gin.Context) {
	a.handleConsultationCommand(c, new(caseworkreq.EscalateConsultation), func(id uint, value any) (caseworkres.ConsultationActionResult, error) {
		return caseWorkService.EscalateConsultation(c.Request.Context(), id, c.GetHeader("Idempotency-Key"), *value.(*caseworkreq.EscalateConsultation))
	}, "升级咨询失败", "咨询已升级")
}

// ResolveConsultation
// @Tags Consultation
// @Summary 当前责任人记录咨询解决结果
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "咨询ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body caseworkreq.ResolveConsultation true "版本、解决结果和后续安排"
// @Success 200 {object} commonres.Response{data=caseworkres.ConsultationActionResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/consultations/{id}/resolve [post]
func (a *CaseWorkApi) ResolveConsultation(c *gin.Context) {
	a.handleConsultationCommand(c, new(caseworkreq.ResolveConsultation), func(id uint, value any) (caseworkres.ConsultationActionResult, error) {
		return caseWorkService.ResolveConsultation(c.Request.Context(), id, c.GetHeader("Idempotency-Key"), *value.(*caseworkreq.ResolveConsultation))
	}, "解决咨询失败", "解决结果已记录")
}

// CloseConsultation
// @Tags Consultation
// @Summary 在解决结果完整后关闭咨询
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "咨询ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body caseworkreq.CloseConsultation true "版本和关闭理由"
// @Success 200 {object} commonres.Response{data=caseworkres.ConsultationActionResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/consultations/{id}/close [post]
func (a *CaseWorkApi) CloseConsultation(c *gin.Context) {
	a.handleConsultationCommand(c, new(caseworkreq.CloseConsultation), func(id uint, value any) (caseworkres.ConsultationActionResult, error) {
		return caseWorkService.CloseConsultation(c.Request.Context(), id, c.GetHeader("Idempotency-Key"), *value.(*caseworkreq.CloseConsultation))
	}, "关闭咨询失败", "咨询已关闭")
}

// ReopenConsultation
// @Tags Consultation
// @Summary 上级重开已关闭咨询并保留历史
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "咨询ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body caseworkreq.ReopenConsultation true "版本和重开原因"
// @Success 200 {object} commonres.Response{data=caseworkres.ConsultationActionResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/consultations/{id}/reopen [post]
func (a *CaseWorkApi) ReopenConsultation(c *gin.Context) {
	a.handleConsultationCommand(c, new(caseworkreq.ReopenConsultation), func(id uint, value any) (caseworkres.ConsultationActionResult, error) {
		return caseWorkService.ReopenConsultation(c.Request.Context(), id, c.GetHeader("Idempotency-Key"), *value.(*caseworkreq.ReopenConsultation))
	}, "重开咨询失败", "咨询已重新打开")
}

func (a *CaseWorkApi) handleConsultationCommand(
	c *gin.Context,
	request any,
	run func(uint, any) (caseworkres.ConsultationActionResult, error),
	fallback string,
	success string,
) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := c.ShouldBindJSON(request); err != nil {
		commonres.FailWithMessage("请求参数无效", c)
		return
	}
	data, err := run(id, request)
	if err != nil {
		handleError(c, err, fallback)
		return
	}
	commonres.OkWithDetailed(data, success, c)
}
