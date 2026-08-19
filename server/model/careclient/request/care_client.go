package request

import (
	"time"

	commonRequest "github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

type CareClientSearch struct {
	commonRequest.PageInfo
	OrganizationID uint   `json:"organizationId" form:"organizationId"`
	Status         string `json:"status" form:"status"`
}

type CreateCareClient struct {
	DisplayCode        string `json:"displayCode"`
	DisplayName        string `json:"displayName"`
	ContactMobile      string `json:"contactMobile"`
	ServiceReason      string `json:"serviceReason"`
	ServicePackageCode string `json:"servicePackageCode"`
	OrganizationID     uint   `json:"organizationId"`
	TeamID             *uint  `json:"teamId"`
	Synthetic          bool   `json:"synthetic"`
}

type UpdateCareClient struct {
	ExpectedVersion    uint   `json:"expectedVersion"`
	DisplayName        string `json:"displayName"`
	ContactMobile      string `json:"contactMobile"`
	ServiceReason      string `json:"serviceReason"`
	ServicePackageCode string `json:"servicePackageCode"`
	TeamID             *uint  `json:"teamId"`
	Status             string `json:"status"`
}

type CreateAssignment struct {
	ExpectedVersion      uint       `json:"expectedVersion"`
	RoleType             string     `json:"roleType"`
	AssigneeID           uint       `json:"assigneeId"`
	TeamID               uint       `json:"teamId"`
	ValidFrom            time.Time  `json:"validFrom"`
	ValidUntil           *time.Time `json:"validUntil"`
	ReplacesAssignmentID *uint      `json:"replacesAssignmentId"`
	Reason               string     `json:"reason"`
}

type CreateConsentRecord struct {
	ExpectedVersion uint      `json:"expectedVersion"`
	ConsentType     string    `json:"consentType"`
	Action          string    `json:"action"`
	TextVersion     string    `json:"textVersion"`
	OccurredAt      time.Time `json:"occurredAt"`
	Source          string    `json:"source"`
	Reason          string    `json:"reason"`
}

type DataLifecycleRequestSearch struct {
	commonRequest.PageInfo
	RequestType string `json:"requestType" form:"requestType"`
}

type CreateDataLifecycleRequest struct {
	ExpectedVersion uint      `json:"expectedVersion" binding:"required"`
	RequestType     string    `json:"requestType" binding:"required"`
	RequestedAt     time.Time `json:"requestedAt" binding:"required"`
	Source          string    `json:"source" binding:"required"`
	Reason          string    `json:"reason" binding:"required,max=1000"`
}
