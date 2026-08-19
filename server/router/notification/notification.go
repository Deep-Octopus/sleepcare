package notification

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type NotificationRouter struct{}

func (r *NotificationRouter) InitNotificationRouter(private, public *gin.RouterGroup) {
	readGroup := private.Group("care")
	writeGroup := private.Group("care").Use(middleware.OperationRecord())
	{
		readGroup.GET("deliveries", notificationApi.ListDeliveries)
		readGroup.GET("notification-provider-readiness", notificationApi.GetProviderReadiness)
	}
	{
		writeGroup.POST("deliveries/:id/resend", notificationApi.Resend)
	}
	public.POST("care/notification-provider-callbacks/:providerCode", notificationApi.ProviderCallback)
}
