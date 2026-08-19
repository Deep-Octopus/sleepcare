package clientaccess

import (
	"net/http"
	"strconv"

	clientmodel "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess"
	commonreq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	commonres "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	supervisionreq "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/request"
	supervisionres "github.com/flipped-aurora/gin-vue-admin/server/model/supervision/response"
	"github.com/gin-gonic/gin"
)

var (
	_ supervisionres.ClientSatisfactionDetail
	_ supervisionres.SubmitSatisfactionResult
)

// ListSatisfactionRequests
// @Tags ClientSatisfaction
// @Summary 获取当前受限会话内的本人服务评价邀请
// @Produce application/json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量，最大100"
// @Param status query string false "邀请状态"
// @Success 200 {object} commonres.Response{data=commonres.PageResult{list=[]supervisionres.ClientSatisfactionSummary},msg=string}
// @Failure 401 {object} commonres.Response
// @Failure 403 {object} commonres.Response
// @Router /care/client/satisfaction-requests [get]
func (a *ClientAccessApi) ListSatisfactionRequests(c *gin.Context) {
	var req supervisionreq.ClientSatisfactionSearch
	if err := c.ShouldBindQuery(&req); err != nil {
		badClientRequest(c, "查询参数无效")
		return
	}
	list, total, err := clientAccessService.ListSatisfactionRequests(c.Request.Context(), req)
	if err != nil {
		handleClientAccessError(c, err, "查询服务评价失败")
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

// GetSatisfactionRequest
// @Tags ClientSatisfaction
// @Summary 获取当前受限会话内的本人服务评价详情
// @Produce application/json
// @Param id path int true "评价邀请ID"
// @Success 200 {object} commonres.Response{data=supervisionres.ClientSatisfactionDetail,msg=string}
// @Failure 401 {object} commonres.Response
// @Failure 403 {object} commonres.Response
// @Router /care/client/satisfaction-requests/{id} [get]
func (a *ClientAccessApi) GetSatisfactionRequest(c *gin.Context) {
	id, ok := clientSatisfactionRequestID(c)
	if !ok {
		return
	}
	data, err := clientAccessService.GetSatisfactionRequest(c.Request.Context(), id)
	if err != nil {
		handleClientAccessError(c, err, "查询服务评价详情失败")
		return
	}
	commonres.OkWithDetailed(data, "查询成功", c)
}

// SubmitSatisfactionResponse
// @Tags ClientSatisfaction
// @Summary 当前受限会话提交本人服务评价
// @Accept application/json
// @Produce application/json
// @Param id path int true "评价邀请ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body supervisionreq.SubmitSatisfactionResponse true "版本、1至5星和补充意见"
// @Success 200 {object} commonres.Response{data=supervisionres.SubmitSatisfactionResult,msg=string}
// @Failure 400 {object} commonres.Response
// @Failure 401 {object} commonres.Response
// @Failure 403 {object} commonres.Response
// @Router /care/client/satisfaction-requests/{id}/responses [post]
func (a *ClientAccessApi) SubmitSatisfactionResponse(c *gin.Context) {
	id, ok := clientSatisfactionRequestID(c)
	if !ok {
		return
	}
	var req supervisionreq.SubmitSatisfactionResponse
	if err := c.ShouldBindJSON(&req); err != nil {
		badClientRequest(c, "请求参数无效")
		return
	}
	data, err := clientAccessService.SubmitSatisfactionResponse(
		c.Request.Context(),
		id,
		c.GetHeader("Idempotency-Key"),
		req,
	)
	if err != nil {
		handleClientAccessError(c, err, "提交服务评价失败")
		return
	}
	commonres.OkWithDetailed(data, "感谢你的反馈", c)
}

func clientSatisfactionRequestID(c *gin.Context) (uint, bool) {
	value, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || value == 0 {
		c.JSON(http.StatusBadRequest, commonres.Response{
			Code: clientmodel.CodeInvalidArgument,
			Data: nil,
			Msg:  "评价邀请ID无效",
		})
		return 0, false
	}
	return uint(value), true
}
