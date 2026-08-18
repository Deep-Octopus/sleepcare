package clientaccess

import v1 "github.com/flipped-aurora/gin-vue-admin/server/api/v1"

type RouterGroup struct {
	ClientAccessRouter
}

var clientAccessApi = v1.ApiGroupApp.ClientAccessApiGroup.ClientAccessApi
