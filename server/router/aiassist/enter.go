package aiassist

import api "github.com/flipped-aurora/gin-vue-admin/server/api/v1"

type RouterGroup struct {
	AIShadowRouter
}

var aiShadowApi = api.ApiGroupApp.AIAssistApiGroup.AIShadowApi
