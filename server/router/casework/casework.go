package casework

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type CaseWorkRouter struct{}

func (r *CaseWorkRouter) InitCaseWorkRouter(router *gin.RouterGroup) {
	writeGroup := router.Group("care").Use(middleware.OperationRecord())
	readGroup := router.Group("care")
	{
		writeGroup.POST("attention-cases/:id/acknowledge", caseWorkApi.AcknowledgeAttentionCase)
		writeGroup.POST("attention-cases/:id/handling-records", caseWorkApi.CreateHandlingRecord)
		writeGroup.POST("attention-cases/:id/escalate", caseWorkApi.EscalateAttentionCase)
		writeGroup.POST("attention-cases/:id/close", caseWorkApi.CloseAttentionCase)
		writeGroup.POST("attention-cases/:id/reopen", caseWorkApi.ReopenAttentionCase)
	}
	{
		readGroup.GET("workbench", caseWorkApi.GetWorkbench)
		readGroup.GET("attention-cases", caseWorkApi.ListAttentionCases)
		readGroup.GET("attention-cases/:id", caseWorkApi.GetAttentionCase)
	}
}
