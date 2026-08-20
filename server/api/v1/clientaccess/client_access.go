package clientaccess

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	clientmodel "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess"
	clientreq "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess/request"
	clientres "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess/response"
	commonres "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/logger"
	"github.com/gin-gonic/gin"
)

type ClientAccessApi struct{}

var _ = clientres.RedeemResult{}

// Login
// @Tags ClientAccess
// @Summary 使用康养用户账号密码建立客户端会话
// @Accept application/json
// @Produce application/json
// @Param data body clientreq.Login true "账号和密码"
// @Success 200 {object} commonres.Response{data=clientres.LoginResult,msg=string}
// @Failure 400 {object} commonres.Response
// @Failure 401 {object} commonres.Response
// @Failure 429 {object} commonres.Response
// @Failure 500 {object} commonres.Response
// @Router /care/client-auth/login [post]
func (a *ClientAccessApi) Login(c *gin.Context) {
	var req clientreq.Login
	if err := c.ShouldBindJSON(&req); err != nil {
		clearClientSessionCookie(c)
		badClientRequest(c, "请输入账号和密码")
		return
	}
	data, sessionToken, err := clientAccessService.Login(c.Request.Context(), req)
	if err != nil {
		clearClientSessionCookie(c)
		handleClientAccessError(c, err, "登录失败")
		return
	}
	setClientSessionCookie(c, sessionToken, data.ExpiresAt)
	commonres.OkWithDetailed(data, "登录成功", c)
}

// Redeem
// @Tags ClientAccess
// @Summary 兑换一次性访问链接并建立受限客户端会话
// @Accept application/json
// @Produce application/json
// @Param data body clientreq.Redeem true "一次性访问凭证"
// @Success 200 {object} commonres.Response{data=clientres.RedeemResult,msg=string}
// @Failure 400 {object} commonres.Response
// @Failure 500 {object} commonres.Response
// @Router /care/client-access/redeem [post]
func (a *ClientAccessApi) Redeem(c *gin.Context) {
	var req clientreq.Redeem
	if err := c.ShouldBindJSON(&req); err != nil {
		badClientRequest(c, "请求参数无效")
		return
	}
	data, sessionToken, err := clientAccessService.Redeem(c.Request.Context(), req.Grant)
	if err != nil {
		handleClientAccessError(c, err, "兑换访问链接失败")
		return
	}
	setClientSessionCookie(c, sessionToken, data.ExpiresAt)
	commonres.OkWithDetailed(data, "访问已建立", c)
}

// GetProfile
// @Tags ClientAccess
// @Summary 获取当前康养用户会话资料
// @Produce application/json
// @Success 200 {object} commonres.Response{data=clientres.SessionProfile,msg=string}
// @Failure 401 {object} commonres.Response
// @Failure 500 {object} commonres.Response
// @Router /care/client/me [get]
func (a *ClientAccessApi) GetProfile(c *gin.Context) {
	data, err := clientAccessService.GetProfile(c.Request.Context())
	if err != nil {
		handleClientAccessError(c, err, "读取个人信息失败")
		return
	}
	commonres.OkWithDetailed(data, "查询成功", c)
}

// Logout
// @Tags ClientAccess
// @Summary 退出当前康养用户会话
// @Produce application/json
// @Success 200 {object} commonres.Response{data=clientres.LogoutResult,msg=string}
// @Failure 401 {object} commonres.Response
// @Failure 500 {object} commonres.Response
// @Router /care/client/logout [post]
func (a *ClientAccessApi) Logout(c *gin.Context) {
	data, err := clientAccessService.Logout(c.Request.Context())
	if err != nil {
		handleClientAccessError(c, err, "退出登录失败")
		return
	}
	clearClientSessionCookie(c)
	commonres.OkWithDetailed(data, "已退出登录", c)
}

func setClientSessionCookie(c *gin.Context, token string, expiresAt time.Time) {
	maxAge, cookieExpiresAt := clientSessionCookieWindow(global.GVA_CONFIG.Care.Now(), expiresAt, time.Now())
	http.SetCookie(c.Writer, &http.Cookie{
		Name: clientSessionCookieName(), Value: token, Path: clientSessionCookiePath(), MaxAge: maxAge, Expires: cookieExpiresAt,
		HttpOnly: true, Secure: global.GVA_CONFIG.Care.ClientAccess.CookieSecure, SameSite: http.SameSiteLaxMode,
	})
}

func clearClientSessionCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: clientSessionCookieName(), Value: "", Path: clientSessionCookiePath(), MaxAge: -1,
		HttpOnly: true, Secure: global.GVA_CONFIG.Care.ClientAccess.CookieSecure, SameSite: http.SameSiteLaxMode,
	})
}

func clientSessionCookieName() string {
	name := strings.TrimSpace(global.GVA_CONFIG.Care.ClientAccess.CookieName)
	if name == "" {
		return "gva_client_session"
	}
	return name
}

func clientSessionCookiePath() string {
	path := strings.TrimSpace(global.GVA_CONFIG.Care.ClientAccess.CookiePath)
	if path == "" {
		return global.GVA_CONFIG.System.RouterPrefix + "/care/client"
	}
	return path
}

func clientSessionCookieWindow(businessNow, sessionExpiresAt, wallNow time.Time) (int, time.Time) {
	remaining := sessionExpiresAt.Sub(businessNow)
	maxAge := int(remaining.Seconds())
	if maxAge < 1 {
		maxAge = 1
	}
	return maxAge, wallNow.Add(time.Duration(maxAge) * time.Second)
}

// ListTasks
// @Tags ClientTasks
// @Summary 获取当前受限会话内的本人任务
// @Produce application/json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量，最大100"
// @Success 200 {object} commonres.Response{data=commonres.PageResult{list=[]clientres.TaskSummary},msg=string}
// @Failure 400 {object} commonres.Response
// @Failure 401 {object} commonres.Response
// @Failure 500 {object} commonres.Response
// @Router /care/client/tasks [get]
func (a *ClientAccessApi) ListTasks(c *gin.Context) {
	var req clientreq.TaskSearch
	if err := c.ShouldBindQuery(&req); err != nil {
		badClientRequest(c, "查询参数无效")
		return
	}
	list, total, err := clientAccessService.ListTasks(c.Request.Context(), req)
	if err != nil {
		handleClientAccessError(c, err, "查询任务失败")
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

// GetTask
// @Tags ClientTasks
// @Summary 获取当前受限会话内的任务详情
// @Produce application/json
// @Param taskId path int true "任务ID"
// @Success 200 {object} commonres.Response{data=clientres.TaskDetail,msg=string}
// @Failure 401 {object} commonres.Response
// @Failure 403 {object} commonres.Response
// @Failure 500 {object} commonres.Response
// @Router /care/client/tasks/{taskId} [get]
func (a *ClientAccessApi) GetTask(c *gin.Context) {
	taskID, ok := clientTaskID(c)
	if !ok {
		return
	}
	data, err := clientAccessService.GetTask(c.Request.Context(), taskID)
	if err != nil {
		handleClientAccessError(c, err, "查询任务详情失败")
		return
	}
	commonres.OkWithDetailed(data, "查询成功", c)
}

// GetQuestionnaire
// @Tags ClientTasks
// @Summary 获取任务冻结问卷和可恢复草稿
// @Produce application/json
// @Param taskId path int true "任务ID"
// @Success 200 {object} commonres.Response{data=clientres.Questionnaire,msg=string}
// @Failure 401 {object} commonres.Response
// @Failure 403 {object} commonres.Response
// @Failure 500 {object} commonres.Response
// @Router /care/client/tasks/{taskId}/questionnaire [get]
func (a *ClientAccessApi) GetQuestionnaire(c *gin.Context) {
	taskID, ok := clientTaskID(c)
	if !ok {
		return
	}
	data, err := clientAccessService.GetQuestionnaire(c.Request.Context(), taskID)
	if err != nil {
		handleClientAccessError(c, err, "查询任务内容失败")
		return
	}
	commonres.OkWithDetailed(data, "查询成功", c)
}

// RecordInteraction
// @Tags ClientTasks
// @Summary 幂等记录打开、确认或开始填写事实
// @Accept application/json
// @Produce application/json
// @Param taskId path int true "任务ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body clientreq.RecordInteraction true "交互类型和任务版本"
// @Success 200 {object} commonres.Response{data=clientres.InteractionResult,msg=string}
// @Failure 400 {object} commonres.Response
// @Failure 401 {object} commonres.Response
// @Failure 403 {object} commonres.Response
// @Failure 500 {object} commonres.Response
// @Router /care/client/tasks/{taskId}/interactions [post]
func (a *ClientAccessApi) RecordInteraction(c *gin.Context) {
	taskID, ok := clientTaskID(c)
	if !ok {
		return
	}
	var req clientreq.RecordInteraction
	if err := c.ShouldBindJSON(&req); err != nil {
		badClientRequest(c, "请求参数无效")
		return
	}
	data, err := clientAccessService.RecordInteraction(c.Request.Context(), taskID, c.GetHeader("Idempotency-Key"), req)
	if err != nil {
		handleClientAccessError(c, err, "记录任务进度失败")
		return
	}
	commonres.OkWithDetailed(data, "任务进度已记录", c)
}

// SaveDraft
// @Tags ClientTasks
// @Summary 幂等保存任务草稿
// @Accept application/json
// @Produce application/json
// @Param taskId path int true "任务ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body clientreq.SaveDraft true "草稿版本和答案"
// @Success 200 {object} commonres.Response{data=clientres.DraftResult,msg=string}
// @Failure 400 {object} commonres.Response
// @Failure 401 {object} commonres.Response
// @Failure 403 {object} commonres.Response
// @Failure 500 {object} commonres.Response
// @Router /care/client/tasks/{taskId}/draft [put]
func (a *ClientAccessApi) SaveDraft(c *gin.Context) {
	taskID, ok := clientTaskID(c)
	if !ok {
		return
	}
	var req clientreq.SaveDraft
	if err := c.ShouldBindJSON(&req); err != nil {
		badClientRequest(c, "请求参数无效")
		return
	}
	data, err := clientAccessService.SaveDraft(c.Request.Context(), taskID, c.GetHeader("Idempotency-Key"), req)
	if err != nil {
		handleClientAccessError(c, err, "保存草稿失败")
		return
	}
	commonres.OkWithDetailed(data, "草稿已保存", c)
}

// SubmitTask
// @Tags ClientTasks
// @Summary 幂等提交任务答案
// @Accept application/json
// @Produce application/json
// @Param taskId path int true "任务ID"
// @Param Idempotency-Key header string true "幂等键"
// @Param data body clientreq.SubmitTask true "任务版本、来源和答案"
// @Success 200 {object} commonres.Response{data=clientres.SubmitResult,msg=string}
// @Failure 400 {object} commonres.Response
// @Failure 401 {object} commonres.Response
// @Failure 403 {object} commonres.Response
// @Failure 500 {object} commonres.Response
// @Router /care/client/tasks/{taskId}/submit [post]
func (a *ClientAccessApi) SubmitTask(c *gin.Context) {
	taskID, ok := clientTaskID(c)
	if !ok {
		return
	}
	var req clientreq.SubmitTask
	if err := c.ShouldBindJSON(&req); err != nil {
		badClientRequest(c, "请求参数无效")
		return
	}
	data, err := clientAccessService.SubmitTask(c.Request.Context(), taskID, c.GetHeader("Idempotency-Key"), req)
	if err != nil {
		handleClientAccessError(c, err, "提交任务失败")
		return
	}
	commonres.OkWithDetailed(data, "已提交，等待处理", c)
}

func clientTaskID(c *gin.Context) (uint, bool) {
	value, err := strconv.ParseUint(c.Param("taskId"), 10, 64)
	if err != nil || value == 0 {
		badClientRequest(c, "任务ID无效")
		return 0, false
	}
	return uint(value), true
}

func handleClientAccessError(c *gin.Context, err error, fallback string) {
	var domainErr *clientmodel.DomainError
	if errors.As(err, &domainErr) {
		status := domainErr.HTTPStatus
		if status == 0 {
			status = http.StatusOK
		}
		c.JSON(status, commonres.Response{Code: domainErr.Code, Data: nil, Msg: domainErr.Message})
		return
	}
	logger.WithCtx(c.Request.Context()).Mod("clientaccess").Err(err).Error(fallback)
	c.JSON(http.StatusInternalServerError, commonres.Response{Code: commonres.ERROR, Data: nil, Msg: fallback})
}

func badClientRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, commonres.Response{Code: clientmodel.CodeInvalidArgument, Data: nil, Msg: message})
}
