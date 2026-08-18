package questionnaire

import "github.com/gin-gonic/gin"

type QuestionnaireRouter struct{}

func (r *QuestionnaireRouter) InitQuestionnaireRouter(router *gin.RouterGroup) {
	readGroup := router.Group("care")
	{
		readGroup.GET("questionnaire-versions", questionnaireApi.ListQuestionnaireVersions)
		readGroup.GET("questionnaire-versions/:id", questionnaireApi.GetQuestionnaireVersion)
	}
}
