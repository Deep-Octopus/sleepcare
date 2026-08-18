package questionnaire

import "fmt"

const (
	CodeInvalidArgument       = 41001
	CodeIdempotencyConflict   = 41002
	CodeVersionConflict       = 41003
	CodeResourceNotFound      = 41004
	CodeOperationNotAllowed   = 41005
	CodeAccessScopeDenied     = 41104
	CodeContentNotPublished   = 41401
	CodeContentDisabled       = 41402
	CodeSubmissionInvalid     = 41403
	CodeRuleExecutionDisabled = 41404
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

func NewForbiddenError(message string) error {
	return &DomainError{Code: CodeAccessScopeDenied, Message: message, HTTPStatus: 403}
}
