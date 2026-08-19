package supervision

import (
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type SupervisionRouter struct{}

func (r *SupervisionRouter) InitSupervisionRouter(router *gin.RouterGroup) {
	readGroup := router.Group("care")
	writeGroup := router.Group("care").Use(middleware.OperationRecord())
	{
		readGroup.GET("daily-summaries", supervisionApi.ListDailySummaries)
		readGroup.GET("daily-summaries/:id", supervisionApi.GetDailySummary)
		readGroup.GET("reviews", supervisionApi.ListReviews)
		readGroup.GET("satisfaction-responses", supervisionApi.ListSatisfactionResponses)
		readGroup.GET("satisfaction-follow-ups", supervisionApi.ListSatisfactionFollowUps)
		readGroup.GET("satisfaction-follow-ups/:id", supervisionApi.GetSatisfactionFollowUp)
	}
	{
		writeGroup.POST("reviews/:id/guidance", supervisionApi.AddGuidance)
		writeGroup.POST("reviews/:id/intervene", supervisionApi.Intervene)
		writeGroup.POST("satisfaction-follow-ups/:id/acknowledge", supervisionApi.AcknowledgeSatisfactionFollowUp)
		writeGroup.POST("satisfaction-follow-ups/:id/resolve", supervisionApi.ResolveSatisfactionFollowUp)
	}
}
