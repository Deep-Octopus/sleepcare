package casework

import api "github.com/flipped-aurora/gin-vue-admin/server/api/v1"

type RouterGroup struct {
	CaseWorkRouter
}

var caseWorkApi = api.ApiGroupApp.CaseWorkApiGroup.CaseWorkApi
