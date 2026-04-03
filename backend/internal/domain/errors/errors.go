package errors

import "fmt"

type Code string

const (
	CodeInvalidArgument  Code = "invalid_argument"
	CodeNotFound         Code = "not_found"
	CodeConflict         Code = "conflict"
	CodeUnauthorized     Code = "unauthorized"
	CodeForbidden        Code = "forbidden"
	CodeExternalFailure  Code = "external_failure"
	CodeInternal         Code = "internal"
	CodePreconditionFail Code = "precondition_failed"
)

type DomainError struct {
	Code    Code
	Message string
	Cause   error
	Details map[string]string
}

func (e *DomainError) Error() string {
	if e.Cause == nil {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
}

func (e *DomainError) Unwrap() error { return e.Cause }

func New(code Code, msg string) *DomainError {
	return &DomainError{Code: code, Message: msg}
}

func Wrap(code Code, msg string, cause error) *DomainError {
	return &DomainError{Code: code, Message: msg, Cause: cause}
}

func NewWithDetails(code Code, msg string, details map[string]string) *DomainError {
	return &DomainError{Code: code, Message: msg, Details: cloneDetails(details)}
}

func WrapWithDetails(code Code, msg string, cause error, details map[string]string) *DomainError {
	return &DomainError{Code: code, Message: msg, Cause: cause, Details: cloneDetails(details)}
}

func (e *DomainError) WithDetails(details map[string]string) *DomainError {
	if e == nil {
		return nil
	}
	e.Details = cloneDetails(details)
	return e
}

func cloneDetails(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
