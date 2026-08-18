package carepath

import api "github.com/flipped-aurora/gin-vue-admin/server/api/v1"

type RouterGroup struct {
	CarePathRouter
}

var carePathApi = api.ApiGroupApp.CarePathApiGroup.CarePathApi
