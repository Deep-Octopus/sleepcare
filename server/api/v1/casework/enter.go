package casework

import "github.com/flipped-aurora/gin-vue-admin/server/service"

type ApiGroup struct {
	CaseWorkApi
}

var caseWorkService = service.ServiceGroupApp.CaseWorkServiceGroup.CaseWorkService
