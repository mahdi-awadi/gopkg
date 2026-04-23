package paymentgateway

import (
	"context"
	"errors"
	"testing"
)

func TestBuildCallbackURLWithReference(t *testing.T) {
	cases := []struct {
		name, url, ref, want string
	}{
		{"plain", "https://x/cb", "abc", "https://x/cb/abc"},
		{"trailing-slash", "https://x/cb/", "abc", "https://x/cb/abc"},
		{"with-query", "https://x/cb?a=1", "abc", "https://x/cb/abc?a=1"},
		{"trailing-slash-and-query", "https://x/cb/?a=1", "abc", "https://x/cb/abc?a=1"},
		{"empty-baseurl", "", "abc", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := BuildCallbackURLWithReference(c.url, c.ref); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestExtractReferenceFromPath(t *testing.T) {
	cases := []struct {
		name, path, want string
		wantErr          bool
	}{
		{"plain", "/cb/abc", "abc", false},
		{"trailing-slash", "/cb/abc/", "abc", false},
		{"full-url-like", "https://x/cb/abc", "abc", false},
		{"with-query", "/cb/abc?x=1", "abc", false},
		{"empty", "", "", true},
		{"slash-only", "/", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ExtractReferenceFromPath(c.path)
			if c.wantErr {
				if !errors.Is(err, ErrInvalidRequest) {
					t.Errorf("want ErrInvalidRequest, got %v", err)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestCallbackURL_RoundTrip(t *testing.T) {
	// A caller-assigned reference should survive build + extract.
	built := BuildCallbackURLWithReference("https://x/cb?session=42", "ref-001")
	got, err := ExtractReferenceFromPath(built)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if got != "ref-001" {
		t.Errorf("got %q, want ref-001", got)
	}
}

// fakeGateway is a compile-time check that the interface is satisfiable
// without importing real HTTP / SDK dependencies.
type fakeGateway struct{}

func (fakeGateway) Code() string                      { return "fake" }
func (fakeGateway) Name() string                      { return "Fake Gateway" }
func (fakeGateway) SupportedCurrencies() []string     { return []string{"USD"} }
func (fakeGateway) SupportedMethods() []string        { return []string{"card"} }
func (fakeGateway) Charge(context.Context, PaymentRequest) (PaymentResponse, error) {
	return PaymentResponse{Status: StatusCaptured}, nil
}
func (fakeGateway) Status(context.Context, string) (StatusReport, error) {
	return StatusReport{Status: StatusCaptured}, nil
}
func (fakeGateway) Cancel(context.Context, string) error { return nil }
func (fakeGateway) Refund(context.Context, RefundRequest) (RefundResponse, error) {
	return RefundResponse{Status: StatusRefunded}, nil
}

// Compile-time interface check.
var _ Gateway = (*fakeGateway)(nil)

func TestFakeGateway_SatisfiesInterface(t *testing.T) {
	var g Gateway = fakeGateway{}
	resp, err := g.Charge(context.Background(), PaymentRequest{Amount: 100, Currency: "USD"})
	if err != nil || resp.Status != StatusCaptured {
		t.Errorf("charge through interface: got (%+v, %v)", resp, err)
	}
}

func TestSentinelErrors_UniqueValues(t *testing.T) {
	// Sanity check — all sentinels are distinct values.
	errs := []error{
		ErrDeclined, ErrInsufficientFunds, ErrInvalidAmount, ErrUnsupportedMethod,
		ErrUnsupportedCurrency, ErrGatewayTimeout, ErrInvalidRequest, ErrNotFound,
	}
	for i, a := range errs {
		for j, b := range errs {
			if i != j && errors.Is(a, b) {
				t.Errorf("sentinels %d and %d compare equal", i, j)
			}
		}
	}
}
