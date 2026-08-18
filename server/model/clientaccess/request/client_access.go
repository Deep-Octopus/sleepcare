package request

import (
	"time"

	commonreq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

type Redeem struct {
	Grant string `json:"grant" binding:"required"`
}

type TaskSearch struct {
	commonreq.PageInfo
}

type RecordInteraction struct {
	ExpectedVersion uint   `json:"expectedVersion" binding:"required"`
	InteractionType string `json:"interactionType" binding:"required"`
}

type SaveDraft struct {
	ExpectedVersion uint           `json:"expectedVersion"`
	Answers         map[string]any `json:"answers" binding:"required"`
}

type SubmitTask struct {
	ExpectedTaskVersion uint           `json:"expectedTaskVersion" binding:"required"`
	Source              string         `json:"source" binding:"required"`
	Answers             map[string]any `json:"answers" binding:"required"`
	ClientOccurredAt    *time.Time     `json:"clientOccurredAt"`
}
