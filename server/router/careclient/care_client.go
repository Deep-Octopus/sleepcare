package careclient

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type CareClientRouter struct{}

func (r *CareClientRouter) InitCareClientRouter(router *gin.RouterGroup) {
	writeGroup := router.Group("care").Use(middleware.OperationRecord())
	readGroup := router.Group("care")
	{
		writeGroup.POST("clients", careClientApi.CreateCareClient)
		writeGroup.PUT("clients/:id", careClientApi.UpdateCareClient)
		writeGroup.POST("clients/:id/assignments", careClientApi.CreateCareAssignment)
		writeGroup.POST("clients/:id/consent-records", careClientApi.CreateCareConsent)
	}
	{
		readGroup.GET("clients", careClientApi.ListCareClients)
		readGroup.GET("clients/:id", careClientApi.GetCareClient)
		readGroup.GET("client-options", careClientApi.GetCareClientOptions)
	}
}
