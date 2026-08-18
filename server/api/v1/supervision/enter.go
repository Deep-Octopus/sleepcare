package supervision

import "github.com/flipped-aurora/gin-vue-admin/server/service"

type ApiGroup struct {
	SupervisionApi
}

var supervisionService = service.ServiceGroupApp.SupervisionServiceGroup.SupervisionService
