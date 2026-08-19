package aiassist

import "github.com/gin-gonic/gin"

type AIShadowRouter struct{}

func (r *AIShadowRouter) InitAIShadowRouter(private *gin.RouterGroup) {
	readGroup := private.Group("care")
	{
		readGroup.GET("ai-shadow-readiness", aiShadowApi.GetShadowReadiness)
	}
}
