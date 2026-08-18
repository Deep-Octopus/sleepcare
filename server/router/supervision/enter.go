package supervision

import api "github.com/flipped-aurora/gin-vue-admin/server/api/v1"

type RouterGroup struct {
	SupervisionRouter
}

var supervisionApi = api.ApiGroupApp.SupervisionApiGroup.SupervisionApi
