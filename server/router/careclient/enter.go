package careclient

import api "github.com/flipped-aurora/gin-vue-admin/server/api/v1"

type RouterGroup struct {
	CareClientRouter
}

var careClientApi = api.ApiGroupApp.CareClientApiGroup.CareClientApi
