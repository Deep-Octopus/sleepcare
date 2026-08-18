package questionnaire

import (
	"errors"
	"net/http"
	"strconv"

	commonres "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	qmodel "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire"
	qreq "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire/request"
	qres "github.com/flipped-aurora/gin-vue-admin/server/model/questionnaire/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/logger"
	"github.com/gin-gonic/gin"
)

type QuestionnaireApi struct{}

// Keep the explicit response package visible to both the Go compiler and swag.
var _ qres.QuestionnaireVersionDetail

// ListQuestionnaireVersions
// @Tags Questionnaire
// @Summary 分页获取问卷版本预览列表
// @Security ApiKeyAuth
// @Produce application/json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量，最大100"
// @Param keyword query string false "编码、版本或标题"
// @Param status query string false "生命周期状态"
// @Param usageScope query string false "使用范围"
// @Param synthetic query bool false "是否合成版本"
// @Success 200 {object} commonres.Response{data=commonres.PageResult{list=[]qres.QuestionnaireVersionSummary},msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/questionnaire-versions [get]
func (a *QuestionnaireApi) ListQuestionnaireVersions(c *gin.Context) {
	var req qreq.QuestionnaireVersionSearch
	if err := c.ShouldBindQuery(&req); err != nil {
		commonres.FailWithMessage("查询参数无效", c)
		return
	}
	list, total, err := questionnaireService.ListVersions(c.Request.Context(), req)
	if err != nil {
		handleQuestionnaireError(c, err, "查询问卷版本失败")
		return
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}
	commonres.OkWithDetailed(commonres.PageResult{List: list, Total: total, Page: req.Page, PageSize: req.PageSize}, "查询成功", c)
}

// GetQuestionnaireVersion
// @Tags Questionnaire
// @Summary 获取问卷、题目和绑定关注规则版本详情
// @Security ApiKeyAuth
// @Produce application/json
// @Param id path int true "问卷版本ID"
// @Success 200 {object} commonres.Response{data=qres.QuestionnaireVersionDetail,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/questionnaire-versions/{id} [get]
func (a *QuestionnaireApi) GetQuestionnaireVersion(c *gin.Context) {
	value, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || value == 0 {
		commonres.FailWithMessage("问卷版本ID无效", c)
		return
	}
	data, err := questionnaireService.GetVersion(c.Request.Context(), uint(value))
	if err != nil {
		handleQuestionnaireError(c, err, "查询问卷版本详情失败")
		return
	}
	commonres.OkWithDetailed(data, "查询成功", c)
}

func handleQuestionnaireError(c *gin.Context, err error, fallback string) {
	var domainErr *qmodel.DomainError
	if errors.As(err, &domainErr) {
		if domainErr.HTTPStatus == http.StatusForbidden {
			commonres.Forbidden(domainErr.Code, domainErr.Message, c)
			return
		}
		c.JSON(http.StatusOK, commonres.Response{Code: domainErr.Code, Data: nil, Msg: domainErr.Message})
		return
	}
	logger.WithCtx(c.Request.Context()).Mod("questionnaire").Err(err).Error(fallback)
	commonres.FailWithMessage(fallback, c)
}
