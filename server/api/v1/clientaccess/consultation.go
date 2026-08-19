package clientaccess

import (
	"net/http"
	"strconv"

	caseworkreq "github.com/flipped-aurora/gin-vue-admin/server/model/casework/request"
	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
	clientmodel "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess"
	commonreq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	commonres "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
)

var (
	_ caseworkres.ConsultationActionResult
	_ caseworkres.ClientConsultationDetail
)

// CreateConsultation
// @Tags ClientConsultations
// @Summary 当前受限会话发起在线咨询
// @Accept application/json
// @Produce application/json
// @Param Idempotency-Key header string true "幂等键"
// @Param data body caseworkreq.CreateConsultation true "咨询主题、问题和联系优先级"
// @Success 200 {object} commonres.Response{data=caseworkres.ConsultationActionResult,msg=string}
// @Failure 400 {object} commonres.Response
// @Failure 401 {object} commonres.Response
// @Failure 403 {object} commonres.Response
// @Router /care/client/consultations [post]
func (a *ClientAccessApi) CreateConsultation(c *gin.Context) {
	var req caseworkreq.CreateConsultation
	if err := c.ShouldBindJSON(&req); err != nil {
		badClientRequest(c, "请求参数无效")
		return
	}
	data, err := clientAccessService.CreateConsultation(c.Request.Context(), c.GetHeader("Idempotency-Key"), req)
	if err != nil {
		handleClientAccessError(c, err, "发起咨询失败")
		return
	}
	commonres.OkWithDetailed(data, "咨询已受理，人工回复时间以服务安排为准", c)
}

// ListConsultations
// @Tags ClientConsultations
// @Summary 获取当前受限会话内的本人咨询
// @Produce application/json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量，最大100"
// @Param status query string false "咨询状态"
// @Success 200 {object} commonres.Response{data=commonres.PageResult{list=[]caseworkres.ClientConsultationSummary},msg=string}
// @Failure 401 {object} commonres.Response
// @Failure 403 {object} commonres.Response
// @Router /care/client/consultations [get]
func (a *ClientAccessApi) ListConsultations(c *gin.Context) {
	var req caseworkreq.ClientConsultationSearch
	if err := c.ShouldBindQuery(&req); err != nil {
		badClientRequest(c, "查询参数无效")
		return
	}
	list, total, err := clientAccessService.ListConsultations(c.Request.Context(), req)
	if err != nil {
		handleClientAccessError(c, err, "查询咨询失败")
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
// @Tags ClientConsultations
// @Summary 获取当前受限会话内的本人咨询详情
// @Produce application/json
// @Param id path int true "咨询ID"
// @Success 200 {object} commonres.Response{data=caseworkres.ClientConsultationDetail,msg=string}
// @Failure 401 {object} commonres.Response
// @Failure 403 {object} commonres.Response
// @Router /care/client/consultations/{id} [get]
func (a *ClientAccessApi) GetConsultation(c *gin.Context) {
	id, ok := clientConsultationID(c)
	if !ok {
		return
	}
	data, err := clientAccessService.GetConsultation(c.Request.Context(), id)
	if err != nil {
		handleClientAccessError(c, err, "查询咨询详情失败")
		return
	}
	commonres.OkWithDetailed(data, "查询成功", c)
}

// AddConsultationMessage
// @Tags ClientConsultations
// @Summary 当前受限会话为本人咨询追加补充信息
// @Accept application/json
// @Produce application/json
// @Param id path int true "咨询ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body caseworkreq.AddClientConsultationMessage true "版本和补充信息"
// @Success 200 {object} commonres.Response{data=caseworkres.ConsultationActionResult,msg=string}
// @Failure 400 {object} commonres.Response
// @Failure 401 {object} commonres.Response
// @Failure 403 {object} commonres.Response
// @Router /care/client/consultations/{id}/messages [post]
func (a *ClientAccessApi) AddConsultationMessage(c *gin.Context) {
	id, ok := clientConsultationID(c)
	if !ok {
		return
	}
	var req caseworkreq.AddClientConsultationMessage
	if err := c.ShouldBindJSON(&req); err != nil {
		badClientRequest(c, "请求参数无效")
		return
	}
	data, err := clientAccessService.AddConsultationMessage(
		c.Request.Context(),
		id,
		c.GetHeader("Idempotency-Key"),
		req,
	)
	if err != nil {
		handleClientAccessError(c, err, "补充咨询信息失败")
		return
	}
	commonres.OkWithDetailed(data, "补充信息已提交", c)
}

func clientConsultationID(c *gin.Context) (uint, bool) {
	value, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || value == 0 {
		c.JSON(http.StatusBadRequest, commonres.Response{Code: clientmodel.CodeInvalidArgument, Data: nil, Msg: "咨询ID无效"})
		return 0, false
	}
	return uint(value), true
}
