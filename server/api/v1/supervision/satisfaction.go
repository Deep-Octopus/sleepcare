package supervision

import (
	commonres "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	supervisionreq "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/request"
	supervisionres "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/response"
	"github.com/gin-gonic/gin"
)

var (
	_ supervisionres.SatisfactionFollowUpDetail
	_ supervisionres.SatisfactionFollowUpActionResult
)

// ListSatisfactionResponses
// @Tags Supervision
// @Summary 查询授权机构内的匿名服务评价响应
// @Security ApiKeyAuth
// @Produce application/json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量，最大100"
// @Param rating query int false "星级，1至5"
// @Param followUpStatus query string false "跟进状态，NONE表示未生成跟进"
// @Success 200 {object} commonres.Response{data=commonres.PageResult{list=[]supervisionres.SatisfactionResponseItem},msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/satisfaction-responses [get]
func (a *SupervisionApi) ListSatisfactionResponses(c *gin.Context) {
	var req supervisionreq.SatisfactionResponseSearch
	if err := c.ShouldBindQuery(&req); err != nil {
		commonres.FailWithMessage("查询参数无效", c)
		return
	}
	list, total, err := supervisionService.ListSatisfactionResponses(c.Request.Context(), req)
	if err != nil {
		handleError(c, err, "查询服务评价失败")
		return
	}
	normalizePage(&req.PageInfo)
	commonres.OkWithDetailed(commonres.PageResult{
		List: list, Total: total, Page: req.Page, PageSize: req.PageSize,
	}, "查询成功", c)
}

// ListSatisfactionFollowUps
// @Tags Supervision
// @Summary 查询授权机构内的低分质量跟进
// @Security ApiKeyAuth
// @Produce application/json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量，最大100"
// @Param status query string false "跟进状态"
// @Success 200 {object} commonres.Response{data=commonres.PageResult{list=[]supervisionres.SatisfactionFollowUpSummary},msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/satisfaction-follow-ups [get]
func (a *SupervisionApi) ListSatisfactionFollowUps(c *gin.Context) {
	var req supervisionreq.SatisfactionFollowUpSearch
	if err := c.ShouldBindQuery(&req); err != nil {
		commonres.FailWithMessage("查询参数无效", c)
		return
	}
	list, total, err := supervisionService.ListSatisfactionFollowUps(c.Request.Context(), req)
	if err != nil {
		handleError(c, err, "查询质量跟进失败")
		return
	}
	normalizePage(&req.PageInfo)
	commonres.OkWithDetailed(commonres.PageResult{
		List: list, Total: total, Page: req.Page, PageSize: req.PageSize,
	}, "查询成功", c)
}

// GetSatisfactionFollowUp
// @Tags Supervision
// @Summary 查询授权机构内的质量跟进详情
// @Security ApiKeyAuth
// @Produce application/json
// @Param id path int true "质量跟进ID"
// @Success 200 {object} commonres.Response{data=supervisionres.SatisfactionFollowUpDetail,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/satisfaction-follow-ups/{id} [get]
func (a *SupervisionApi) GetSatisfactionFollowUp(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	data, err := supervisionService.GetSatisfactionFollowUp(c.Request.Context(), id)
	if err != nil {
		handleError(c, err, "查询质量跟进详情失败")
		return
	}
	commonres.OkWithDetailed(data, "查询成功", c)
}

// AcknowledgeSatisfactionFollowUp
// @Tags Supervision
// @Summary 当前责任上级接收低分质量跟进
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "质量跟进ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body supervisionreq.AcknowledgeSatisfactionFollowUp true "版本和接收说明"
// @Success 200 {object} commonres.Response{data=supervisionres.SatisfactionFollowUpActionResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/satisfaction-follow-ups/{id}/acknowledge [post]
func (a *SupervisionApi) AcknowledgeSatisfactionFollowUp(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req supervisionreq.AcknowledgeSatisfactionFollowUp
	if err := c.ShouldBindJSON(&req); err != nil {
		commonres.FailWithMessage("请求参数无效", c)
		return
	}
	data, err := supervisionService.AcknowledgeSatisfactionFollowUp(
		c.Request.Context(),
		id,
		c.GetHeader("Idempotency-Key"),
		req,
	)
	if err != nil {
		handleError(c, err, "接收质量跟进失败")
		return
	}
	commonres.OkWithDetailed(data, "质量跟进已接收", c)
}

// ResolveSatisfactionFollowUp
// @Tags Supervision
// @Summary 当前责任上级解决低分质量跟进
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "质量跟进ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body supervisionreq.ResolveSatisfactionFollowUp true "版本、核查结果、改进动作和使用边界确认"
// @Success 200 {object} commonres.Response{data=supervisionres.SatisfactionFollowUpActionResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/satisfaction-follow-ups/{id}/resolve [post]
func (a *SupervisionApi) ResolveSatisfactionFollowUp(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req supervisionreq.ResolveSatisfactionFollowUp
	if err := c.ShouldBindJSON(&req); err != nil {
		commonres.FailWithMessage("请求参数无效", c)
		return
	}
	data, err := supervisionService.ResolveSatisfactionFollowUp(
		c.Request.Context(),
		id,
		c.GetHeader("Idempotency-Key"),
		req,
	)
	if err != nil {
		handleError(c, err, "解决质量跟进失败")
		return
	}
	commonres.OkWithDetailed(data, "质量跟进已解决", c)
}
