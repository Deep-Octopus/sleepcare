package clientaccess

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type ClientAccessRouter struct{}

func (r *ClientAccessRouter) InitClientAccessRouter(public *gin.RouterGroup) {
	public.POST("care/client-access/redeem", clientAccessApi.Redeem)

	client := public.Group("care/client")
	client.Use(middleware.ClientSessionAuth())
	client.GET("tasks", clientAccessApi.ListTasks)
	client.GET("tasks/:taskId", clientAccessApi.GetTask)
	client.GET("tasks/:taskId/questionnaire", clientAccessApi.GetQuestionnaire)

	write := client.Group("")
	write.Use(middleware.ClientSameOrigin())
	write.POST("tasks/:taskId/interactions", clientAccessApi.RecordInteraction)
	write.PUT("tasks/:taskId/draft", clientAccessApi.SaveDraft)
	write.POST("tasks/:taskId/submit", clientAccessApi.SubmitTask)
}
