package middleware

import (
	"net/http"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	clientmodel "github.com/flipped-aurora/gin-vue-admin/server/model/clientaccess"
	commonres "github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/service"
	clientservice "github.com/flipped-aurora/gin-vue-admin/server/service/clientaccess"
	"github.com/gin-gonic/gin"
)

func ClientSessionAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		name := clientCookieName()
		token, err := c.Cookie(name)
		if err != nil || strings.TrimSpace(token) == "" {
			abortClientSession(c)
			return
		}
		identity, err := service.ServiceGroupApp.ClientAccessServiceGroup.ClientAccessService.Authenticate(c.Request.Context(), token)
		if err != nil {
			clearClientCookie(c, name)
			abortClientSession(c)
			return
		}
		c.Request = c.Request.WithContext(clientservice.ContextWithSessionIdentity(c.Request.Context(), identity))
		c.Next()
	}
}

func ClientSameOrigin() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		allowed := false
		for _, configured := range global.GVA_CONFIG.Care.ClientAccess.AllowedOrigins {
			if origin != "" && origin == strings.TrimSpace(configured) {
				allowed = true
				break
			}
		}
		if !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, commonres.Response{
				Code: clientmodel.CodeAccessScopeDenied, Data: nil, Msg: "请求来源不受信任",
			})
			return
		}
		c.Next()
	}
}

func clientCookieName() string {
	name := strings.TrimSpace(global.GVA_CONFIG.Care.ClientAccess.CookieName)
	if name == "" {
		return "gva_client_session"
	}
	return name
}

func clientCookiePath() string {
	path := strings.TrimSpace(global.GVA_CONFIG.Care.ClientAccess.CookiePath)
	if path == "" {
		return global.GVA_CONFIG.System.RouterPrefix + "/care/client"
	}
	return path
}

func abortClientSession(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, commonres.Response{
		Code: clientmodel.CodeSessionInvalid, Data: nil, Msg: "访问会话无效或已失效",
	})
}

func clearClientCookie(c *gin.Context, name string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: name, Value: "", Path: clientCookiePath(), MaxAge: -1,
		HttpOnly: true, Secure: global.GVA_CONFIG.Care.ClientAccess.CookieSecure, SameSite: http.SameSiteLaxMode,
	})
}
