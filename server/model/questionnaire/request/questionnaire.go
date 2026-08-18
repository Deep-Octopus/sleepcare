package request

import commonreq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"

type QuestionnaireVersionSearch struct {
	commonreq.PageInfo
	Status     string `json:"status" form:"status"`
	UsageScope string `json:"usageScope" form:"usageScope"`
	Synthetic  *bool  `json:"synthetic" form:"synthetic"`
}
