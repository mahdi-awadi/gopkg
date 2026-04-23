package errorsx

import (
	"errors"
	"net/http"
	"testing"
)

func TestNew_AndKindOf(t *testing.T) {
	err := New(KindNotFound, "user missing")
	if KindOf(err) != KindNotFound {
		t.Fatalf("got %v, want KindNotFound", KindOf(err))
	}
}

func TestNewf(t *testing.T) {
	err := Newf(KindInvalidArgument, "field %q is required", "email")
	if err.Error() != `field "email" is required` {
		t.Fatalf("got %q", err.Error())
	}
}

func TestWrap_NilReturnsNil(t *testing.T) {
	if Wrap(KindNotFound, nil, "x") != nil {
		t.Fatal("Wrap(nil) should return nil")
	}
}

func TestWrap_PreservesUnderlying(t *testing.T) {
	underlying := errors.New("db: row not found")
	wrapped := Wrap(KindNotFound, underlying, "fetching user 42")

	if KindOf(wrapped) != KindNotFound {
		t.Fatalf("KindOf=%v", KindOf(wrapped))
	}
	if !errors.Is(wrapped, underlying) {
		t.Fatal("errors.Is should find underlying")
	}
}

func TestIs_MatchesBySentinelKind(t *testing.T) {
	err := New(KindNotFound, "nope")
	if !errors.Is(err, ErrNotFound) {
		t.Fatal("expected errors.Is to match KindNotFound sentinel")
	}
	if errors.Is(err, ErrConflict) {
		t.Fatal("should NOT match different kind")
	}
}

func TestHTTPStatus_Mapping(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{New(KindNotFound, "x"), http.StatusNotFound},
		{New(KindInvalidArgument, "x"), http.StatusBadRequest},
		{New(KindConflict, "x"), http.StatusConflict},
		{New(KindUnauthenticated, "x"), http.StatusUnauthorized},
		{New(KindPermissionDenied, "x"), http.StatusForbidden},
		{New(KindFailedPrecondition, "x"), http.StatusPreconditionFailed},
		{New(KindResourceExhausted, "x"), http.StatusTooManyRequests},
		{New(KindUnavailable, "x"), http.StatusServiceUnavailable},
		{New(KindDeadlineExceeded, "x"), http.StatusGatewayTimeout},
		{errors.New("plain"), http.StatusInternalServerError},
		{nil, http.StatusInternalServerError},
	}
	for _, c := range cases {
		if got := HTTPStatus(c.err); got != c.want {
			t.Errorf("HTTPStatus(%v) = %d, want %d", c.err, got, c.want)
		}
	}
}

func TestKindOf_PlainErrorIsUnknown(t *testing.T) {
	if KindOf(errors.New("x")) != KindUnknown {
		t.Fatal("expected KindUnknown for plain error")
	}
	if KindOf(nil) != KindUnknown {
		t.Fatal("expected KindUnknown for nil")
	}
}
