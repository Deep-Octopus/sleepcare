package notification

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type NotificationRouter struct{}

func (r *NotificationRouter) InitNotificationRouter(router *gin.RouterGroup) {
	readGroup := router.Group("care")
	writeGroup := router.Group("care").Use(middleware.OperationRecord())
	{
		readGroup.GET("deliveries", notificationApi.ListDeliveries)
	}
	{
		writeGroup.POST("deliveries/:id/resend", notificationApi.Resend)
	}
}
