package notification

import "github.com/flipped-aurora/gin-vue-admin/server/service"

type ApiGroup struct {
	NotificationApi
}

var notificationService = service.ServiceGroupApp.NotificationServiceGroup.NotificationService
