package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func runReq(h http.Handler, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func TestRecover_CatchesPanic(t *testing.T) {
	var captured any
	mw := Recover(func(err any, stack []byte, r *http.Request) { captured = err })
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { panic("boom") }))
	rec := runReq(h, "GET", "/", nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if captured != "boom" {
		t.Fatalf("panic not captured, got %v", captured)
	}
}

func TestRequestID_GeneratesWhenAbsent(t *testing.T) {
	var observed string
	h := RequestID("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observed = RequestIDFromContext(r.Context())
	}))
	rec := runReq(h, "GET", "/", nil)
	if observed == "" {
		t.Fatal("ID should be generated")
	}
	if rec.Header().Get(HeaderRequestID) != observed {
		t.Fatal("response header should echo ID")
	}
}

func TestRequestID_ReusesInbound(t *testing.T) {
	var observed string
	h := RequestID("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observed = RequestIDFromContext(r.Context())
	}))
	runReq(h, "GET", "/", map[string]string{HeaderRequestID: "my-id-123"})
	if observed != "my-id-123" {
		t.Fatalf("expected inbound ID, got %q", observed)
	}
}

func TestLogger_CapturesStatusAndDuration(t *testing.T) {
	var entry LogEntry
	h := Logger(func(e LogEntry) { entry = e })(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("short"))
	}))
	runReq(h, "GET", "/teapot", nil)
	if entry.Status != http.StatusTeapot {
		t.Fatalf("status=%d", entry.Status)
	}
	if entry.Method != "GET" || entry.Path != "/teapot" {
		t.Fatalf("method/path wrong: %+v", entry)
	}
	if entry.BytesOut != 5 {
		t.Fatalf("bytes=%d, want 5", entry.BytesOut)
	}
}

func TestChain_Ordering(t *testing.T) {
	var trace []string
	mkLog := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				trace = append(trace, name+":in")
				next.ServeHTTP(w, r)
				trace = append(trace, name+":out")
			})
		}
	}

	h := Chain(mkLog("A"), mkLog("B"))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		trace = append(trace, "handler")
	}))
	runReq(h, "GET", "/", nil)

	want := []string{"A:in", "B:in", "handler", "B:out", "A:out"}
	if strings.Join(trace, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v, want %v", trace, want)
	}
}

func TestCORS_PreflightResponds204(t *testing.T) {
	h := CORS(CORSConfig{AllowOrigins: []string{"*"}})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not be called on OPTIONS")
	}))
	rec := runReq(h, "OPTIONS", "/", map[string]string{"Origin": "https://example.com"})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Fatal("expected CORS method header")
	}
}

func TestCORS_AllowedOrigin(t *testing.T) {
	h := CORS(CORSConfig{AllowOrigins: []string{"https://good.com"}})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := runReq(h, "GET", "/", map[string]string{"Origin": "https://good.com"})
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://good.com" {
		t.Fatal("allowed origin should be echoed")
	}

	rec = runReq(h, "GET", "/", map[string]string{"Origin": "https://bad.com"})
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("bad origin should NOT be echoed")
	}
}

func TestTimeout_RespondsWhenHandlerHangs(t *testing.T) {
	h := Timeout(20 * time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done() // observe timeout
		// Writing after deadline is a no-op for TimeoutHandler; it already wrote 504.
	}))
	start := time.Now()
	rec := runReq(h, "GET", "/", nil)
	if time.Since(start) > 500*time.Millisecond {
		t.Fatal("Timeout took too long")
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 Service Unavailable from TimeoutHandler, got %d", rec.Code)
	}
}
