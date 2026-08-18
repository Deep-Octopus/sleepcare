package supervision

import (
	"errors"
	"net/http"
	"strconv"

	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
	commonreq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	commonres "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	supervisionmodel "github.com/flipped-aurora/gin-vue-admin/server/model/supervision"
	supervisionreq "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/request"
	supervisionres "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/logger"
	"github.com/gin-gonic/gin"
)

type SupervisionApi struct{}

var (
	_ supervisionres.DailySummaryDetail
	_ supervisionres.ReviewItem
	_ caseworkres.ActionResult
)

// ListDailySummaries
// @Tags Supervision
// @Summary 查询今日实时预览与授权范围历史日报版本
// @Security ApiKeyAuth
// @Produce application/json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量，最大100"
// @Param businessDate query string false "业务日期，YYYY-MM-DD"
// @Success 200 {object} commonres.Response{data=commonres.PageResult{list=[]supervisionres.DailySummary},msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/daily-summaries [get]
func (a *SupervisionApi) ListDailySummaries(c *gin.Context) {
	var req supervisionreq.DailySummarySearch
	if err := c.ShouldBindQuery(&req); err != nil {
		commonres.FailWithMessage("查询参数无效", c)
		return
	}
	list, total, err := supervisionService.ListDailySummaries(c.Request.Context(), req)
	if err != nil {
		handleError(c, err, "查询每日汇总失败")
		return
	}
	normalizePage(&req.PageInfo)
	commonres.OkWithDetailed(commonres.PageResult{
		List: list, Total: total, Page: req.Page, PageSize: req.PageSize,
	}, "查询成功", c)
}

// GetDailySummary
// @Tags Supervision
// @Summary 查询授权范围历史日报版本详情
// @Security ApiKeyAuth
// @Produce application/json
// @Param id path int true "日报版本ID"
// @Success 200 {object} commonres.Response{data=supervisionres.DailySummaryDetail,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/daily-summaries/{id} [get]
func (a *SupervisionApi) GetDailySummary(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	data, err := supervisionService.GetDailySummary(c.Request.Context(), id)
	if err != nil {
		handleError(c, err, "查询日报详情失败")
		return
	}
	commonres.OkWithDetailed(data, "查询成功", c)
}

// ListReviews
// @Tags Supervision
// @Summary 查询授权管理范围内的上级复核队列
// @Security ApiKeyAuth
// @Produce application/json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量，最大100"
// @Success 200 {object} commonres.Response{data=commonres.PageResult{list=[]supervisionres.ReviewItem},msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/reviews [get]
func (a *SupervisionApi) ListReviews(c *gin.Context) {
	var req supervisionreq.ReviewSearch
	if err := c.ShouldBindQuery(&req); err != nil {
		commonres.FailWithMessage("查询参数无效", c)
		return
	}
	list, total, err := supervisionService.ListReviews(c.Request.Context(), req)
	if err != nil {
		handleError(c, err, "查询待复核事项失败")
		return
	}
	normalizePage(&req.PageInfo)
	commonres.OkWithDetailed(commonres.PageResult{
		List: list, Total: total, Page: req.Page, PageSize: req.PageSize,
	}, "查询成功", c)
}

// AddGuidance
// @Tags Supervision
// @Summary 追加上级指导或讨论安排
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "复核事项ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body supervisionreq.Guidance true "指导、责任医护、截止时间和事项版本"
// @Success 200 {object} commonres.Response{data=caseworkres.ActionResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/reviews/{id}/guidance [post]
func (a *SupervisionApi) AddGuidance(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req supervisionreq.Guidance
	if err := c.ShouldBindJSON(&req); err != nil {
		commonres.FailWithMessage("请求参数无效", c)
		return
	}
	data, err := supervisionService.AddGuidance(c.Request.Context(), id, c.GetHeader("Idempotency-Key"), req)
	if err != nil {
		handleError(c, err, "追加上级指导失败")
		return
	}
	commonres.OkWithDetailed(data, "指导已追加", c)
}

// Intervene
// @Tags Supervision
// @Summary 上级直接介入并指定责任医护与截止时间
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "复核事项ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body supervisionreq.Intervene true "介入结果、责任医护、截止时间和事项版本"
// @Success 200 {object} commonres.Response{data=caseworkres.ActionResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/reviews/{id}/intervene [post]
func (a *SupervisionApi) Intervene(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req supervisionreq.Intervene
	if err := c.ShouldBindJSON(&req); err != nil {
		commonres.FailWithMessage("请求参数无效", c)
		return
	}
	data, err := supervisionService.Intervene(c.Request.Context(), id, c.GetHeader("Idempotency-Key"), req)
	if err != nil {
		handleError(c, err, "记录上级介入失败")
		return
	}
	commonres.OkWithDetailed(data, "介入已记录", c)
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
	var domainErr *supervisionmodel.DomainError
	if errors.As(err, &domainErr) {
		if domainErr.HTTPStatus == http.StatusForbidden {
			commonres.Forbidden(domainErr.Code, domainErr.Message, c)
			return
		}
		c.JSON(http.StatusOK, commonres.Response{Code: domainErr.Code, Data: nil, Msg: domainErr.Message})
		return
	}
	logger.WithCtx(c.Request.Context()).Mod("supervision").Err(err).Error(fallback)
	commonres.FailWithMessage(fallback, c)
}
