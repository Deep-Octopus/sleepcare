package carepath

import "github.com/flipped-aurora/gin-vue-admin/server/service"

type ApiGroup struct {
	CarePathApi
}

var carePathService = service.ServiceGroupApp.CarePathServiceGroup.CarePathService
