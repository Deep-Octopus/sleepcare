package notification

import api "github.com/flipped-aurora/gin-vue-admin/server/api/v1"

type RouterGroup struct {
	NotificationRouter
}

var notificationApi = api.ApiGroupApp.NotificationApiGroup.NotificationApi
