package request

import commonreq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"

type DeliverySearch struct {
	commonreq.PageInfo
	Status string `json:"status" form:"status"`
}

type Resend struct {
	ExpectedVersion uint   `json:"expectedVersion"`
	Reason          string `json:"reason"`
}
