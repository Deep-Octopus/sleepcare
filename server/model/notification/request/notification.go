package request

import (
	"time"

	commonreq "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

type DeliverySearch struct {
	commonreq.PageInfo
	Status string `json:"status" form:"status"`
}

type Resend struct {
	ExpectedVersion uint   `json:"expectedVersion"`
	Reason          string `json:"reason"`
}

type ProviderCallback struct {
	EventID           string    `json:"eventId"`
	ProviderMessageID string    `json:"providerMessageId"`
	Status            string    `json:"status"`
	OccurredAt        time.Time `json:"occurredAt"`
	FailureCode       string    `json:"failureCode"`
}

type ProviderCallbackSignature struct {
	Timestamp string
	Nonce     string
	Signature string
}
