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
	}
	{
		writeGroup.POST("reviews/:id/guidance", supervisionApi.AddGuidance)
		writeGroup.POST("reviews/:id/intervene", supervisionApi.Intervene)
	}
}
