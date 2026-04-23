// Package errorsx provides a small taxonomy of error kinds that map
// cleanly to HTTP status codes and gRPC status codes, with sentinel
// values matchable via errors.Is / errors.As.
//
// Zero third-party deps.
package errorsx

import (
	"errors"
	"fmt"
	"net/http"
)

// Kind classifies an error into one of a small set of buckets used for
// HTTP/gRPC response mapping.
type Kind int

const (
	// KindUnknown is the default (internal error).
	KindUnknown Kind = iota
	KindNotFound
	KindInvalidArgument
	KindConflict
	KindUnauthenticated
	KindPermissionDenied
	KindFailedPrecondition
	KindResourceExhausted
	KindUnavailable
	KindDeadlineExceeded
)

// Sentinels — use with errors.Is.
var (
	ErrNotFound           = &Error{kind: KindNotFound, msg: "not found"}
	ErrInvalidArgument    = &Error{kind: KindInvalidArgument, msg: "invalid argument"}
	ErrConflict           = &Error{kind: KindConflict, msg: "conflict"}
	ErrUnauthenticated    = &Error{kind: KindUnauthenticated, msg: "unauthenticated"}
	ErrPermissionDenied   = &Error{kind: KindPermissionDenied, msg: "permission denied"}
	ErrFailedPrecondition = &Error{kind: KindFailedPrecondition, msg: "failed precondition"}
	ErrResourceExhausted  = &Error{kind: KindResourceExhausted, msg: "resource exhausted"}
	ErrUnavailable        = &Error{kind: KindUnavailable, msg: "unavailable"}
	ErrDeadlineExceeded   = &Error{kind: KindDeadlineExceeded, msg: "deadline exceeded"}
)

// Error is the canonical error type with a typed Kind.
type Error struct {
	kind Kind
	msg  string
	wrap error
}

// New creates a typed error.
func New(kind Kind, msg string) *Error {
	return &Error{kind: kind, msg: msg}
}

// Newf is like New with fmt.Sprintf.
func Newf(kind Kind, format string, args ...any) *Error {
	return &Error{kind: kind, msg: fmt.Sprintf(format, args...)}
}

// Wrap attaches a Kind to an underlying error. Returns nil if err is nil.
func Wrap(kind Kind, err error, msg string) *Error {
	if err == nil {
		return nil
	}
	return &Error{kind: kind, msg: msg, wrap: err}
}

// KindOf returns the Kind of err (KindUnknown if not an *Error).
// Works through errors.As unwrap chains.
func KindOf(err error) Kind {
	var e *Error
	if errors.As(err, &e) {
		return e.kind
	}
	return KindUnknown
}

// Kind returns the typed kind.
func (e *Error) Kind() Kind { return e.kind }

// Error implements error.
func (e *Error) Error() string {
	if e.wrap != nil {
		return fmt.Sprintf("%s: %s", e.msg, e.wrap.Error())
	}
	return e.msg
}

// Unwrap supports errors.As / errors.Is on wrapped errors.
func (e *Error) Unwrap() error { return e.wrap }

// Is supports errors.Is on the sentinels. Two *Error are equivalent if
// they share a Kind (so errors.Is(someErr, ErrNotFound) is true when
// someErr's Kind is KindNotFound).
func (e *Error) Is(target error) bool {
	var t *Error
	if !errors.As(target, &t) {
		return false
	}
	return e.kind == t.kind
}

// HTTPStatus maps a Kind to an HTTP status code.
func HTTPStatus(err error) int {
	switch KindOf(err) {
	case KindNotFound:
		return http.StatusNotFound
	case KindInvalidArgument:
		return http.StatusBadRequest
	case KindConflict:
		return http.StatusConflict
	case KindUnauthenticated:
		return http.StatusUnauthorized
	case KindPermissionDenied:
		return http.StatusForbidden
	case KindFailedPrecondition:
		return http.StatusPreconditionFailed
	case KindResourceExhausted:
		return http.StatusTooManyRequests
	case KindUnavailable:
		return http.StatusServiceUnavailable
	case KindDeadlineExceeded:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}
