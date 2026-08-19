package notification

import "fmt"

const (
	CodeInvalidArgument        = 41001
	CodeIdempotencyConflict    = 41002
	CodeVersionConflict        = 41003
	CodeResourceNotFound       = 41004
	CodeOperationNotAllowed    = 41005
	CodeAccessScopeDenied      = 41104
	CodeCareAssignmentRequired = 41201
	CodeNotificationFinalized  = 41601
	CodeDeliveryEventInvalid   = 41602
	CodeProviderGateClosed     = 41603
	CodeRetryLimitExceeded     = 41604
	CodeRateLimitExceeded      = 41605
	CodeCostLimitExceeded      = 41606
	CodeCallbackUnauthorized   = 41607
	CodeCallbackReplay         = 41608
	CodeProviderReceiptMissing = 41609
	CodeProviderConfigInvalid  = 41610
)

type DomainError struct {
	Code       int
	Message    string
	HTTPStatus int
}

func (e *DomainError) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

func NewDomainError(code int, message string) error {
	return &DomainError{Code: code, Message: message}
}

func NewForbiddenError(message string) error {
	return &DomainError{Code: CodeAccessScopeDenied, Message: message, HTTPStatus: 403}
}
