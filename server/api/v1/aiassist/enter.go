package aiassist

import "github.com/flipped-aurora/gin-vue-admin/server/service"

type ApiGroup struct {
	AIShadowApi
}

var aiShadowService = service.ServiceGroupApp.AIAssistServiceGroup.AIShadowService
