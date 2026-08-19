package careclient

import "fmt"

const (
	CodeInvalidArgument         = 41001
	CodeIdempotencyConflict     = 41002
	CodeVersionConflict         = 41003
	CodeResourceNotFound        = 41004
	CodeOperationNotAllowed     = 41005
	CodeAccessScopeDenied       = 41104
	CodeCareAssignmentRequired  = 41201
	CodeCareClientUnavailable   = 41202
	CodeDataGovernanceDisabled  = 41901
	CodeLifecycleRequestInvalid = 41902
)

type DomainError struct {
	Code       int
	Message    string
	HTTPStatus int
}

func (e *DomainError) Error() string { return fmt.Sprintf("%d: %s", e.Code, e.Message) }

func NewDomainError(code int, message string) error {
	return &DomainError{Code: code, Message: message}
}

func NewForbiddenError(code int, message string) error {
	return &DomainError{Code: code, Message: message, HTTPStatus: 403}
}
