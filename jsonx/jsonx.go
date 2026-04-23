// Package jsonx provides small helpers for JSON over net/http.
//
// The functions here cover ~80% of an HTTP handler's JSON plumbing:
// decode the request body into a struct (with size limits + unknown-field
// rejection), and write a typed response with the right Content-Type and
// status code.
//
// Zero third-party deps.
package jsonx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// DefaultMaxBodySize caps the size of request bodies accepted by Decode.
// Override via DecodeOptions.MaxBodySize.
const DefaultMaxBodySize = 1 << 20 // 1 MiB

// DecodeOptions controls Decode behavior.
type DecodeOptions struct {
	// MaxBodySize limits the request body (in bytes). 0 → DefaultMaxBodySize.
	MaxBodySize int64
	// DisallowUnknownFields causes Decode to fail when the JSON contains
	// keys not present in the target struct. Recommended for strict APIs.
	DisallowUnknownFields bool
}

// ErrTooLarge is returned by Decode when the body exceeds MaxBodySize.
var ErrTooLarge = errors.New("jsonx: request body too large")

// Decode reads r.Body into dst with the given options. Returns a typed
// error for size overflow (ErrTooLarge) and a wrapped error for all
// other decode failures.
//
// Callers should respond with 400 on a non-nil error.
func Decode(r *http.Request, dst any, opts DecodeOptions) error {
	max := opts.MaxBodySize
	if max <= 0 {
		max = DefaultMaxBodySize
	}
	body := http.MaxBytesReader(nil, r.Body, max)
	defer body.Close()

	dec := json.NewDecoder(body)
	if opts.DisallowUnknownFields {
		dec.DisallowUnknownFields()
	}
	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return ErrTooLarge
		}
		return fmt.Errorf("jsonx: decode: %w", err)
	}
	// Ensure there's no trailing JSON (e.g. a second object).
	var extra json.RawMessage
	if err := dec.Decode(&extra); err != io.EOF {
		return fmt.Errorf("jsonx: trailing content in body")
	}
	return nil
}

// Write renders v as JSON to w with the given status code. Sets
// Content-Type to application/json and a UTF-8 charset.
//
// Returns the encoder error (rare for well-formed structs) so callers
// can log it.
func Write(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// Error is a small helper: sets Content-Type, writes status, and renders
// {"error": message}.
func Error(w http.ResponseWriter, status int, message string) {
	_ = Write(w, status, map[string]string{"error": message})
}
