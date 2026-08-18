package questionnaire

import api "github.com/flipped-aurora/gin-vue-admin/server/api/v1"

type RouterGroup struct {
	QuestionnaireRouter
}

var questionnaireApi = api.ApiGroupApp.QuestionnaireApiGroup.QuestionnaireApi
