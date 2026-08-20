package clientaccess

import "fmt"

const (
	CodeInvalidArgument        = 41001
	CodeIdempotencyConflict    = 41002
	CodeVersionConflict        = 41003
	CodeOperationNotAllowed    = 41005
	CodeGrantInvalid           = 41101
	CodeGrantExpired           = 41102
	CodeSessionInvalid         = 41103
	CodeAccessScopeDenied      = 41104
	CodeCredentialsInvalid     = 41105
	CodeCredentialLocked       = 41106
	CodeCareAssignmentRequired = 41201
	CodeTaskNotOpen            = 41302
	CodeTaskExpired            = 41303
	CodeTaskCancelled          = 41304
	CodeContentNotPublished    = 41401
	CodeContentDisabled        = 41402
	CodeSubmissionInvalid      = 41403
	CodeRuleExecution          = 41404
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

func NewHTTPError(code, status int, message string) error {
	return &DomainError{Code: code, Message: message, HTTPStatus: status}
}
