package request

import (
	commonreq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

type ConsultationSearch struct {
	commonreq.PageInfo
	Status     string `json:"status" form:"status"`
	Urgency    string `json:"urgency" form:"urgency"`
	AssigneeID uint   `json:"assigneeId" form:"assigneeId"`
}

type ClientConsultationSearch struct {
	commonreq.PageInfo
	Status string `json:"status" form:"status"`
}

type CreateConsultation struct {
	Subject string `json:"subject" binding:"required,max=120"`
	Message string `json:"message" binding:"required,max=4000"`
	Urgency string `json:"urgency" binding:"required,oneof=ROUTINE EXPEDITED"`
}

type AddClientConsultationMessage struct {
	ExpectedVersion uint   `json:"expectedVersion" binding:"required"`
	Message         string `json:"message" binding:"required,max=4000"`
}

type AssignConsultation struct {
	ExpectedVersion  uint   `json:"expectedVersion" binding:"required"`
	TargetAssigneeID uint   `json:"targetAssigneeId" binding:"required"`
	TargetRole       string `json:"targetRole" binding:"required,oneof=CARE_STEWARD CLINICIAN"`
	Reason           string `json:"reason" binding:"required,max=2000"`
}

type ReplyConsultation struct {
	ExpectedVersion uint   `json:"expectedVersion" binding:"required"`
	Message         string `json:"message" binding:"required,max=4000"`
	NextStatus      string `json:"nextStatus" binding:"required,oneof=HANDLING WAITING_CLIENT"`
}

type TransferConsultation struct {
	ExpectedVersion  uint   `json:"expectedVersion" binding:"required"`
	TargetAssigneeID uint   `json:"targetAssigneeId" binding:"required"`
	TargetRole       string `json:"targetRole" binding:"required,oneof=CARE_STEWARD CLINICIAN"`
	Reason           string `json:"reason" binding:"required,max=2000"`
}

type EscalateConsultation struct {
	ExpectedVersion  uint   `json:"expectedVersion" binding:"required"`
	TargetAssigneeID uint   `json:"targetAssigneeId" binding:"required"`
	Reason           string `json:"reason" binding:"required,max=2000"`
}

type ResolveConsultation struct {
	ExpectedVersion uint   `json:"expectedVersion" binding:"required"`
	Resolution      string `json:"resolution" binding:"required,max=4000"`
	FollowUpPlan    string `json:"followUpPlan" binding:"omitempty,max=2000"`
}

type CloseConsultation struct {
	ExpectedVersion uint   `json:"expectedVersion" binding:"required"`
	CloseReason     string `json:"closeReason" binding:"required,max=2000"`
}

type ReopenConsultation struct {
	ExpectedVersion uint   `json:"expectedVersion" binding:"required"`
	Reason          string `json:"reason" binding:"required,max=2000"`
}
