// Package paymentgateway defines the canonical payment-gateway
// contract and a small set of domain-neutral request/response types.
//
// This package is contract-only — no concrete gateway adapters ship
// here. Each integration (Stripe, Adyen, local rail, mock) lives in
// its own module and implements Gateway.
//
// Zero third-party dependencies.
package paymentgateway

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Status is the payment life-cycle state surfaced to callers.
// Gateway-specific status codes map onto this set.
type Status string

const (
	// StatusPending — created but not yet captured by the gateway.
	StatusPending Status = "pending"
	// StatusAuthorized — gateway reserved the funds but capture hasn't run.
	StatusAuthorized Status = "authorized"
	// StatusCaptured — funds moved successfully.
	StatusCaptured Status = "captured"
	// StatusFailed — a terminal failure (declined, gateway error, etc).
	StatusFailed Status = "failed"
	// StatusCancelled — voided before capture.
	StatusCancelled Status = "cancelled"
	// StatusRefunded — full or partial refund completed.
	StatusRefunded Status = "refunded"
)

// Known sentinel errors. Gateways SHOULD wrap these with %w for their
// implementation-specific detail so callers can match with errors.Is.
var (
	ErrDeclined          = errors.New("payment declined by gateway")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrInvalidAmount     = errors.New("invalid amount")
	ErrUnsupportedMethod = errors.New("payment method not supported")
	ErrUnsupportedCurrency = errors.New("currency not supported")
	ErrGatewayTimeout    = errors.New("gateway timeout")
	ErrInvalidRequest    = errors.New("invalid request")
	ErrNotFound          = errors.New("payment not found")
)

// PaymentRequest is the caller-side input to a charge.
//
// Minor units: Amount is expressed in the currency's minor unit (cents,
// fils, etc.). For currencies without minor units (JPY, IQD, …) that
// equals the major unit. This matches stripe-style API conventions.
type PaymentRequest struct {
	// Reference is the caller's idempotent identifier for this charge.
	// Gateways MUST echo it into PaymentResponse.Reference and into
	// any callback payload.
	Reference string

	// Amount in the minor unit of Currency.
	Amount int64

	// Currency is the ISO-4217 three-letter code, uppercase.
	Currency string

	// Method is the caller-declared payment method (card, wallet,
	// bank_transfer, …). Gateway interprets.
	Method string

	// Description is a short, human-readable memo passed to the gateway.
	Description string

	// CallbackURL is where the gateway should POST/redirect on completion.
	// Some gateways append query params; use BuildCallbackURLWithReference
	// to make sure your Reference survives the round-trip.
	CallbackURL string

	// Metadata lets callers attach free-form tags that the gateway is
	// expected to preserve (stripe-style). Nil is fine.
	Metadata map[string]string
}

// PaymentResponse is the gateway's reply to a charge attempt.
type PaymentResponse struct {
	// Reference echoes PaymentRequest.Reference.
	Reference string

	// GatewayPaymentID is the gateway's own identifier. Persist this
	// for later status polls / refunds.
	GatewayPaymentID string

	// Status is the current life-cycle stage.
	Status Status

	// RedirectURL is non-empty when the caller must send the end-user
	// through a hosted payment page (3DS, wallet authorization, …).
	RedirectURL string

	// FailureReason is the gateway's text explanation when Status is
	// Failed or Declined. Empty otherwise.
	FailureReason string

	// Raw is the untyped gateway payload for callers that need to
	// inspect provider-specific fields. Gateways MAY leave this nil.
	Raw any
}

// RefundRequest captures a refund attempt against a previous payment.
//
// Amount zero means "refund the full captured amount" (partial refunds
// set Amount > 0).
type RefundRequest struct {
	GatewayPaymentID string
	Amount           int64  // minor units; 0 = full refund
	Currency         string // ISO-4217
	Reason           string // human-readable, for the gateway ledger
	Reference        string // caller's idempotent refund ID
}

// RefundResponse is the gateway's reply to a refund attempt.
type RefundResponse struct {
	Reference         string
	GatewayRefundID   string
	Status            Status
	RefundedAmount    int64
	FailureReason     string
	Raw               any
}

// StatusReport is the output of a status poll.
type StatusReport struct {
	GatewayPaymentID string
	Status           Status
	CapturedAmount   int64
	Currency         string
	FailureReason    string
	Raw              any
}

// GatewayConfig is the untyped representation of a gateway's runtime
// configuration. Keeping this opaque lets consumers feed the same
// struct through their Partner/tenant-scoped secret store without
// imposing a shape here.
type GatewayConfig map[string]any

// Gateway is the contract every payment-gateway adapter implements.
//
// Every I/O-bearing method honors ctx cancellation and returns typed
// errors wrapping the sentinels above where applicable.
type Gateway interface {
	// Code returns the short gateway identifier (e.g. "stripe").
	Code() string
	// Name returns the human-readable name.
	Name() string
	// SupportedCurrencies returns the ISO-4217 codes this gateway
	// accepts. Upper-case, sorted, stable.
	SupportedCurrencies() []string
	// SupportedMethods returns the method identifiers this gateway
	// accepts (e.g. "card", "wallet").
	SupportedMethods() []string

	// Charge initiates a payment. Returns the current Status in the
	// response; callers must not assume capture is complete.
	Charge(ctx context.Context, req PaymentRequest) (PaymentResponse, error)

	// Status fetches the current status of a previously-initiated
	// payment. Returns ErrNotFound if the gateway doesn't recognize
	// the ID.
	Status(ctx context.Context, gatewayPaymentID string) (StatusReport, error)

	// Cancel voids a pending/authorized payment before capture.
	// Returns ErrNotFound if the ID is unknown; wraps ErrInvalidRequest
	// if the payment is already captured.
	Cancel(ctx context.Context, gatewayPaymentID string) error

	// Refund returns a captured payment to the payer. Partial refunds
	// are allowed by setting Amount > 0.
	Refund(ctx context.Context, req RefundRequest) (RefundResponse, error)
}

// BuildCallbackURLWithReference appends a caller-provided reference to
// a callback URL as a path segment, preserving any existing query
// string. Useful when a gateway's callback doesn't round-trip your
// own identifiers — stuff them in the URL and read them on return.
//
//	BuildCallbackURLWithReference("https://x/cb", "abc")       = "https://x/cb/abc"
//	BuildCallbackURLWithReference("https://x/cb?a=1", "abc")   = "https://x/cb/abc?a=1"
//	BuildCallbackURLWithReference("", "abc")                   = ""
func BuildCallbackURLWithReference(baseURL, reference string) string {
	if baseURL == "" {
		return ""
	}
	// Split off the query string first, so trailing-slash trimming
	// operates on the path only ("https://x/cb/?a=1" must produce
	// "https://x/cb/<ref>?a=1", not "https://x/cb//<ref>?a=1").
	path, query := baseURL, ""
	if i := strings.Index(baseURL, "?"); i != -1 {
		path, query = baseURL[:i], baseURL[i:]
	}
	path = strings.TrimSuffix(path, "/")
	return fmt.Sprintf("%s/%s%s", path, reference, query)
}

// ExtractReferenceFromPath returns the last non-empty path segment of
// a URL path. Companion to BuildCallbackURLWithReference.
func ExtractReferenceFromPath(urlPath string) (string, error) {
	p := strings.TrimRight(urlPath, "/")
	if p == "" {
		return "", fmt.Errorf("%w: empty path", ErrInvalidRequest)
	}
	// Strip query string if caller passed a full URL.
	if i := strings.Index(p, "?"); i != -1 {
		p = p[:i]
	}
	if i := strings.LastIndex(p, "/"); i != -1 {
		p = p[i+1:]
	}
	if p == "" {
		return "", fmt.Errorf("%w: no trailing segment", ErrInvalidRequest)
	}
	return p, nil
}
