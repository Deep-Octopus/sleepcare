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
		writeGroup.POST("consultations/:id/assign", caseWorkApi.AssignConsultation)
		writeGroup.POST("consultations/:id/replies", caseWorkApi.ReplyConsultation)
		writeGroup.POST("consultations/:id/transfer", caseWorkApi.TransferConsultation)
		writeGroup.POST("consultations/:id/escalate", caseWorkApi.EscalateConsultation)
		writeGroup.POST("consultations/:id/resolve", caseWorkApi.ResolveConsultation)
		writeGroup.POST("consultations/:id/close", caseWorkApi.CloseConsultation)
		writeGroup.POST("consultations/:id/reopen", caseWorkApi.ReopenConsultation)
	}
	{
		readGroup.GET("workbench", caseWorkApi.GetWorkbench)
		readGroup.GET("attention-cases", caseWorkApi.ListAttentionCases)
		readGroup.GET("attention-cases/:id", caseWorkApi.GetAttentionCase)
		readGroup.GET("consultations", caseWorkApi.ListConsultations)
		readGroup.GET("consultations/:id", caseWorkApi.GetConsultation)
		readGroup.GET("consultations/:id/assignee-options", caseWorkApi.ListConsultationAssigneeOptions)
	}
}
