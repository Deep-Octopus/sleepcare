package request

import commonreq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"

type DailySummarySearch struct {
	commonreq.PageInfo
	BusinessDate string `json:"businessDate" form:"businessDate"`
}

type ReviewSearch struct {
	commonreq.PageInfo
}
