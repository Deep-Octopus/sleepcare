package careclient

import (
	"errors"
	"net/http"
	"strconv"

	caremodel "github.com/flipped-aurora/gin-vue-admin/server/model/careclient"
	carereq "github.com/flipped-aurora/gin-vue-admin/server/model/careclient/request"
	careres "github.com/flipped-aurora/gin-vue-admin/server/model/careclient/response"
	commonres "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/logger"
	"github.com/gin-gonic/gin"
)

type CareClientApi struct{}

// Keep the explicit response package visible to both the Go compiler and swag.
var _ careres.ActionResult

// ListCareClients
// @Tags CareClient
// @Summary 分页获取授权范围内的康养用户
// @Security ApiKeyAuth
// @Produce application/json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量，最大100"
// @Param keyword query string false "显示编码或名称"
// @Param organizationId query int false "机构部门ID"
// @Param status query string false "状态"
// @Success 200 {object} commonres.Response{data=commonres.PageResult{list=[]careres.CareClientSummary},msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/clients [get]
func (a *CareClientApi) ListCareClients(c *gin.Context) {
	var req carereq.CareClientSearch
	if err := c.ShouldBindQuery(&req); err != nil {
		commonres.FailWithMessage("查询参数无效", c)
		return
	}
	list, total, err := careClientService.List(c.Request.Context(), req)
	if err != nil {
		handleError(c, err, "查询康养用户失败")
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

// GetCareClient
// @Tags CareClient
// @Summary 获取授权范围内的康养用户详情
// @Security ApiKeyAuth
// @Produce application/json
// @Param id path int true "康养用户ID"
// @Success 200 {object} commonres.Response{data=careres.CareClientDetail,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/clients/{id} [get]
func (a *CareClientApi) GetCareClient(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	data, err := careClientService.Get(c.Request.Context(), id)
	if err != nil {
		handleError(c, err, "查询康养用户详情失败")
		return
	}
	commonres.OkWithDetailed(data, "查询成功", c)
}

// CreateCareClient
// @Tags CareClient
// @Summary 新建康养用户
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param Idempotency-Key header string true "幂等键"
// @Param data body carereq.CreateCareClient true "康养用户公开资料"
// @Success 200 {object} commonres.Response{data=careres.ActionResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/clients [post]
func (a *CareClientApi) CreateCareClient(c *gin.Context) {
	var req carereq.CreateCareClient
	if err := c.ShouldBindJSON(&req); err != nil {
		commonres.FailWithMessage("请求参数无效", c)
		return
	}
	data, err := careClientService.Create(c.Request.Context(), c.GetHeader("Idempotency-Key"), req)
	if err != nil {
		handleError(c, err, "新建康养用户失败")
		return
	}
	commonres.OkWithDetailed(data, "新建成功", c)
}

// UpdateCareClient
// @Tags CareClient
// @Summary 更新康养用户公开资料
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "康养用户ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body carereq.UpdateCareClient true "公开资料和乐观锁版本"
// @Success 200 {object} commonres.Response{data=careres.ActionResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/clients/{id} [put]
func (a *CareClientApi) UpdateCareClient(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req carereq.UpdateCareClient
	if err := c.ShouldBindJSON(&req); err != nil {
		commonres.FailWithMessage("请求参数无效", c)
		return
	}
	data, err := careClientService.Update(c.Request.Context(), id, c.GetHeader("Idempotency-Key"), req)
	if err != nil {
		handleError(c, err, "更新康养用户失败")
		return
	}
	commonres.OkWithDetailed(data, "更新成功", c)
}

// CreateCareAssignment
// @Tags CareClient
// @Summary 新增或转交责任关系
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "康养用户ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body carereq.CreateAssignment true "责任关系"
// @Success 200 {object} commonres.Response{data=careres.ActionResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/clients/{id}/assignments [post]
func (a *CareClientApi) CreateCareAssignment(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req carereq.CreateAssignment
	if err := c.ShouldBindJSON(&req); err != nil {
		commonres.FailWithMessage("请求参数无效", c)
		return
	}
	data, err := careClientService.CreateAssignment(c.Request.Context(), id, c.GetHeader("Idempotency-Key"), req)
	if err != nil {
		handleError(c, err, "记录责任关系失败")
		return
	}
	commonres.OkWithDetailed(data, "责任关系已记录", c)
}

// CreateCareConsent
// @Tags CareClient
// @Summary 追加固定测试授权或撤回记录
// @Security ApiKeyAuth
// @Accept application/json
// @Produce application/json
// @Param id path int true "康养用户ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body carereq.CreateConsentRecord true "授权事实"
// @Success 200 {object} commonres.Response{data=careres.ActionResult,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/clients/{id}/consent-records [post]
func (a *CareClientApi) CreateCareConsent(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req carereq.CreateConsentRecord
	if err := c.ShouldBindJSON(&req); err != nil {
		commonres.FailWithMessage("请求参数无效", c)
		return
	}
	data, err := careClientService.CreateConsent(c.Request.Context(), id, c.GetHeader("Idempotency-Key"), req)
	if err != nil {
		handleError(c, err, "记录授权事实失败")
		return
	}
	commonres.OkWithDetailed(data, "授权事实已记录", c)
}

// GetCareClientOptions
// @Tags CareClient
// @Summary 获取康养用户维护选项
// @Security ApiKeyAuth
// @Produce application/json
// @Success 200 {object} commonres.Response{data=careres.ClientOptions,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/client-options [get]
func (a *CareClientApi) GetCareClientOptions(c *gin.Context) {
	data, err := careClientService.Options(c.Request.Context())
	if err != nil {
		handleError(c, err, "查询维护选项失败")
		return
	}
	commonres.OkWithDetailed(data, "查询成功", c)
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
	var domainErr *caremodel.DomainError
	if errors.As(err, &domainErr) {
		if domainErr.HTTPStatus == http.StatusForbidden {
			commonres.Forbidden(domainErr.Code, domainErr.Message, c)
			return
		}
		c.JSON(http.StatusOK, commonres.Response{Code: domainErr.Code, Data: nil, Msg: domainErr.Message})
		return
	}
	logger.WithCtx(c.Request.Context()).Mod("careclient").Err(err).Error(fallback)
	commonres.FailWithMessage(fallback, c)
}
