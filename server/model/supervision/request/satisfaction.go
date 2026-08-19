package request

import commonreq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"

type ClientSatisfactionSearch struct {
	commonreq.PageInfo
	Status string `json:"status" form:"status"`
}

type SubmitSatisfactionResponse struct {
	ExpectedVersion uint   `json:"expectedVersion" binding:"required"`
	Rating          uint8  `json:"rating" binding:"required,min=1,max=5"`
	Comment         string `json:"comment" binding:"omitempty,max=2000"`
}

type SatisfactionResponseSearch struct {
	commonreq.PageInfo
	Rating         uint8  `json:"rating" form:"rating" binding:"omitempty,min=1,max=5"`
	FollowUpStatus string `json:"followUpStatus" form:"followUpStatus"`
}

type SatisfactionFollowUpSearch struct {
	commonreq.PageInfo
	Status string `json:"status" form:"status"`
}

type AcknowledgeSatisfactionFollowUp struct {
	ExpectedVersion uint   `json:"expectedVersion" binding:"required"`
	Note            string `json:"note" binding:"required,max=2000"`
}

type ResolveSatisfactionFollowUp struct {
	ExpectedVersion        uint   `json:"expectedVersion" binding:"required"`
	Resolution             string `json:"resolution" binding:"required,max=4000"`
	ImprovementAction      string `json:"improvementAction" binding:"omitempty,max=2000"`
	UsageBoundaryConfirmed bool   `json:"usageBoundaryConfirmed" binding:"required"`
}
