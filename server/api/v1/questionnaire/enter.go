package questionnaire

import "github.com/flipped-aurora/gin-vue-admin/server/service"

type ApiGroup struct {
	QuestionnaireApi
}

var questionnaireService = service.ServiceGroupApp.QuestionnaireServiceGroup.QuestionnaireService
