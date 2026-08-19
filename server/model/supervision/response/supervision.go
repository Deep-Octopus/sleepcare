package response

import (
	"time"

	caseworkres "github.com/flipped-aurora/gin-vue-admin/server/model/casework/response"
)

type DailySummary struct {
	ID                      uint       `json:"id"`
	BusinessDate            string     `json:"businessDate"`
	SummaryType             string     `json:"summaryType"`
	Version                 *uint      `json:"version"`
	MetricDefinitionVersion string     `json:"metricDefinitionVersion"`
	GenerationType          string     `json:"generationType"`
	GeneratedAt             *time.Time `json:"generatedAt"`
	SourceCutoffAt          *time.Time `json:"sourceCutoffAt"`
	PreviousVersionID       *uint      `json:"previousVersionId"`
	CorrectionReason        string     `json:"correctionReason"`
	IsLatest                bool       `json:"isLatest"`
	ServedClients           int64      `json:"servedClients"`
	DueTasks                int64      `json:"dueTasks"`
	SubmittedTasks          int64      `json:"submittedTasks"`
	OverdueTasks            int64      `json:"overdueTasks"`
	DeliveryIssues          int64      `json:"deliveryIssues"`
	OpenAttentionCases      int64      `json:"openAttentionCases"`
	ResolvedAttentionCases  int64      `json:"resolvedAttentionCases"`
	ConsultationsOpened     int64      `json:"consultationsOpened"`
	ConsultationsClosed     int64      `json:"consultationsClosed"`
	OpenConsultations       int64      `json:"openConsultations"`
	OpenTodos               int64      `json:"openTodos"`
	ReviewRequired          int64      `json:"reviewRequired"`
}

type DailySummaryDetail struct {
	DailySummary
	FocusCases        []caseworkres.AttentionCaseSummary `json:"focusCases"`
	RevisionChanges   []MetricChange                     `json:"revisionChanges"`
	FocusCasesChanged bool                               `json:"focusCasesChanged"`
}

type MetricChange struct {
	Key    string `json:"key"`
	Before int64  `json:"before"`
	After  int64  `json:"after"`
}

type DashboardCoverage struct {
	RequestedPastDays int      `json:"requestedPastDays"`
	SnapshotDays      int      `json:"snapshotDays"`
	RevisedDates      int      `json:"revisedDates"`
	MissingDates      []string `json:"missingDates"`
}

type OperationsDashboard struct {
	AsOf                    time.Time         `json:"asOf"`
	TimeZone                string            `json:"timeZone"`
	UsageScope              string            `json:"usageScope"`
	FormalReportingEnabled  bool              `json:"formalReportingEnabled"`
	AttributionPolicyStatus string            `json:"attributionPolicyStatus"`
	MetricDefinitionVersion string            `json:"metricDefinitionVersion"`
	Current                 DailySummary      `json:"current"`
	RecentSnapshots         []DailySummary    `json:"recentSnapshots"`
	Coverage                DashboardCoverage `json:"coverage"`
}

type ReviewItem struct {
	ID              uint      `json:"id"`
	AttentionCaseID uint      `json:"attentionCaseId"`
	Status          string    `json:"status"`
	RequestedAt     time.Time `json:"requestedAt"`
	RequestedBy     uint      `json:"requestedBy"`
}
