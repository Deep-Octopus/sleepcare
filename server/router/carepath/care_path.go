package carepath

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type CarePathRouter struct{}

func (r *CarePathRouter) InitCarePathRouter(router *gin.RouterGroup) {
	writeGroup := router.Group("care").Use(middleware.OperationRecord())
	readGroup := router.Group("care")
	{
		writeGroup.POST("clients/:id/plan-previews", carePathApi.PreviewPlan)
		writeGroup.POST("clients/:id/plan-instances", carePathApi.StartPlan)
		writeGroup.POST("plan-instances/:id/pause", carePathApi.PausePlan)
		writeGroup.POST("plan-instances/:id/resume", carePathApi.ResumePlan)
	}
	{
		readGroup.GET("plan-versions", carePathApi.ListPlanVersions)
		readGroup.GET("plan-versions/:id", carePathApi.GetPlanVersion)
		readGroup.GET("clients/:id/plan-instances", carePathApi.ListClientPlans)
		readGroup.GET("tasks", carePathApi.ListTasks)
		readGroup.GET("tasks/:id", carePathApi.GetTask)
	}
}
