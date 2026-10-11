// Package apperr defines errors that carry enough detail for clients to
// react to them: a kind for the transport to map, a stable machine-readable
// code, a human-readable message and optionally the offending field.
//
// Domain packages declare their sentinel errors with the constructors below,
// so errors.Is keeps working on the sentinel pointers.
package apperr

import "errors"

// Kind classifies an error independently of any transport.
type Kind int

const (
	// KindNotFound means a requested resource does not exist.
	KindNotFound Kind = iota + 1
	// KindInvalid means the request is malformed or violates a validation rule.
	KindInvalid
	// KindConflict means the request clashes with existing state.
	KindConflict
)

// Error is an error with a kind, a stable code and an optional field name.
type Error struct {
	Kind    Kind
	Code    string
	Message string
	Field   string
}

func (e *Error) Error() string {
	return e.Message
}

// NotFound builds an error for a missing resource.
func NotFound(code, message string) *Error {
	return &Error{Kind: KindNotFound, Code: code, Message: message}
}

// Invalid builds a validation error. The field may be empty.
func Invalid(code, message, field string) *Error {
	return &Error{Kind: KindInvalid, Code: code, Message: message, Field: field}
}

// Conflict builds an error for a clash with existing state. The field may be empty.
func Conflict(code, message, field string) *Error {
	return &Error{Kind: KindConflict, Code: code, Message: message, Field: field}
}

// As finds the first *Error in err's chain.
func As(err error) (*Error, bool) {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
