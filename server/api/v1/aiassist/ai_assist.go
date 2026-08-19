package aiassist

import (
	"errors"
	"net/http"

	aiassistmodel "github.com/flipped-aurora/gin-vue-admin/server/model/aiassist"
	aiassistres "github.com/flipped-aurora/gin-vue-admin/server/model/aiassist/response"
	commonres "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/logger"
	"github.com/gin-gonic/gin"
)

type AIShadowApi struct{}

var _ aiassistres.ShadowReadiness

// GetShadowReadiness
// @Tags AIAssist
// @Summary 获取工作人员 AI 影子能力关闭态与阻塞项
// @Security ApiKeyAuth
// @Produce application/json
// @Success 200 {object} commonres.Response{data=aiassistres.ShadowReadiness,msg=string}
// @Failure 403 {object} commonres.Response
// @Router /care/ai-shadow-readiness [get]
func (a *AIShadowApi) GetShadowReadiness(c *gin.Context) {
	data, err := aiShadowService.GetShadowReadiness(c.Request.Context())
	if err != nil {
		handleError(c, err, "查询 AI 影子能力准备状态失败")
		return
	}
	commonres.OkWithDetailed(data, "查询成功", c)
}

func handleError(c *gin.Context, err error, fallback string) {
	var domainErr *aiassistmodel.DomainError
	if errors.As(err, &domainErr) {
		if domainErr.HTTPStatus == http.StatusForbidden {
			commonres.Forbidden(domainErr.Code, domainErr.Message, c)
			return
		}
		if domainErr.HTTPStatus != 0 {
			c.JSON(domainErr.HTTPStatus, commonres.Response{Code: domainErr.Code, Data: nil, Msg: domainErr.Message})
			return
		}
		c.JSON(http.StatusOK, commonres.Response{Code: domainErr.Code, Data: nil, Msg: domainErr.Message})
		return
	}
	logger.WithCtx(c.Request.Context()).Mod("aiassist").Err(err).Error(fallback)
	commonres.FailWithMessage(fallback, c)
}
