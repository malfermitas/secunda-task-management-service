// Package errs defines the shared error contract used across bounded contexts.
package errs

// Kind describes the high-level category of an error.
type Kind int

const (
	KindInternal Kind = iota
	KindValidation
	KindUnauthorized
	KindForbidden
	KindNotFound
	KindConflict
)

// Error is a typed application error with a stable machine-readable code.
type Error struct {
	Code    string
	Message string
	Kind    Kind
}

func (e *Error) Error() string {
	return e.Message
}

// New creates a typed error.
func New(code, message string, kind Kind) *Error {
	return &Error{Code: code, Message: message, Kind: kind}
}

// Validation creates a validation error.
func Validation(message string) *Error {
	return New("validation_error", message, KindValidation)
}

// InvalidCredentials creates a generic invalid credentials error.
func InvalidCredentials() *Error {
	return New("invalid_credentials", "invalid email or password", KindUnauthorized)
}

// Forbidden creates a permission error.
func Forbidden(message string) *Error {
	return New("forbidden", message, KindForbidden)
}

// NotFound creates a not found error.
func NotFound(code, message string) *Error {
	return New(code, message, KindNotFound)
}

// Conflict creates a conflict error.
func Conflict(code, message string) *Error {
	return New(code, message, KindConflict)
}
