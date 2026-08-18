package request

import (
	"time"

	commonreq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

type PlanVersionSearch struct {
	commonreq.PageInfo
	Status     string `json:"status" form:"status"`
	UsageScope string `json:"usageScope" form:"usageScope"`
	Synthetic  *bool  `json:"synthetic" form:"synthetic"`
}

type PreviewPlan struct {
	PlanTemplateVersionID uint      `json:"planTemplateVersionId"`
	AnchorAt              time.Time `json:"anchorAt"`
}

type StartPlan struct {
	ExpectedClientVersion uint   `json:"expectedClientVersion"`
	PreviewID             string `json:"previewId"`
}

type PlanStateAction struct {
	ExpectedVersion uint   `json:"expectedVersion"`
	Reason          string `json:"reason"`
}

type TaskSearch struct {
	commonreq.PageInfo
	CareClientID    uint   `json:"careClientId" form:"careClientId"`
	PlanInstanceID  uint   `json:"planInstanceId" form:"planInstanceId"`
	ExecutionStatus string `json:"executionStatus" form:"executionStatus"`
	TimingStatus    string `json:"timingStatus" form:"timingStatus"`
	DayCode         string `json:"dayCode" form:"dayCode"`
}
