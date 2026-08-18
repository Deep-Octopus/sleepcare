package careclient

import "github.com/flipped-aurora/gin-vue-admin/server/service"

type ApiGroup struct {
	CareClientApi
}

var careClientService = service.ServiceGroupApp.CareClientServiceGroup.CareClientService
