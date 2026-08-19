package aiassist

import "fmt"

const CodeAccessScopeDenied = 41104

type DomainError struct {
	Code       int
	Message    string
	HTTPStatus int
}

func (e *DomainError) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

func NewForbiddenError(message string) error {
	return &DomainError{Code: CodeAccessScopeDenied, Message: message, HTTPStatus: 403}
}
