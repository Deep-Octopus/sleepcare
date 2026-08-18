package clientaccess

import "github.com/flipped-aurora/gin-vue-admin/server/service"

type ApiGroup struct {
	ClientAccessApi
}

var clientAccessService = service.ServiceGroupApp.ClientAccessServiceGroup.ClientAccessService
