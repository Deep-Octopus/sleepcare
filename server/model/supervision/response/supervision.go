package response

import (
	"time"

	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
)

type DailySummary struct {
	ID                     uint   `json:"id"`
	BusinessDate           string `json:"businessDate"`
	SummaryType            string `json:"summaryType"`
	Version                *uint  `json:"version"`
	ServedClients          int64  `json:"servedClients"`
	DueTasks               int64  `json:"dueTasks"`
	SubmittedTasks         int64  `json:"submittedTasks"`
	DeliveryIssues         int64  `json:"deliveryIssues"`
	OpenAttentionCases     int64  `json:"openAttentionCases"`
	ResolvedAttentionCases int64  `json:"resolvedAttentionCases"`
	ReviewRequired         int64  `json:"reviewRequired"`
}

type DailySummaryDetail struct {
	DailySummary
	FocusCases []caseworkres.AttentionCaseSummary `json:"focusCases"`
}

type ReviewItem struct {
	ID              uint      `json:"id"`
	AttentionCaseID uint      `json:"attentionCaseId"`
	Status          string    `json:"status"`
	RequestedAt     time.Time `json:"requestedAt"`
	RequestedBy     uint      `json:"requestedBy"`
}
