// Package middleware provides small, composable net/http middlewares:
// Recover, RequestID, Logger, Timeout, CORS.
//
// Use Chain to compose in order of execution (outermost first):
//
//	handler := middleware.Chain(
//	    middleware.Recover(nil),
//	    middleware.RequestID(""),
//	    middleware.Timeout(5*time.Second),
//	)(yourHandler)
//
// Zero third-party deps.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

// Middleware wraps one http.Handler, returning another.
type Middleware func(http.Handler) http.Handler

// Chain composes middlewares; the first-listed wraps outermost (runs first).
func Chain(mws ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		h := next
		for i := len(mws) - 1; i >= 0; i-- {
			h = mws[i](h)
		}
		return h
	}
}

// ─── Recover ─────────────────────────────────────────────────────────────────

// PanicLogger is the optional callback Recover uses. If nil, panics are
// silently written to the response with 500.
type PanicLogger func(err any, stack []byte, r *http.Request)

// Recover catches panics in downstream handlers, logs them via logFn
// (if non-nil), and responds with 500. The client never sees a crashed
// connection.
func Recover(logFn PanicLogger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					if logFn != nil {
						logFn(err, debug.Stack(), r)
					}
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// ─── RequestID ───────────────────────────────────────────────────────────────

// requestIDKey is a private context key type.
type requestIDKey struct{}

// HeaderRequestID is the HTTP header name for request IDs.
const HeaderRequestID = "X-Request-Id"

// RequestID returns a middleware that reuses the inbound X-Request-Id
// header if present, otherwise generates a random 16-char hex string.
// The value is stored on ctx (retrievable via RequestIDFromContext) and
// echoed on the response.
//
// headerName overrides the header name (empty → HeaderRequestID).
func RequestID(headerName string) Middleware {
	if headerName == "" {
		headerName = HeaderRequestID
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(headerName)
			if id == "" {
				id = newRequestID()
			}
			w.Header().Set(headerName, id)
			ctx := context.WithValue(r.Context(), requestIDKey{}, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequestIDFromContext returns the request ID for a context, or "" if unset.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey{}).(string); ok {
		return v
	}
	return ""
}

func newRequestID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Extremely unlikely; fall back to a timestamp fragment.
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

// ─── Logger ──────────────────────────────────────────────────────────────────

// LogEntry is one completed HTTP request.
type LogEntry struct {
	Method     string
	Path       string
	Status     int
	Duration   time.Duration
	BytesOut   int
	RequestID  string
	RemoteAddr string
}

// LogFunc is the callback invoked by Logger once per request.
type LogFunc func(LogEntry)

// Logger records method/path/status/duration/size and invokes logFn
// on every request.
func Logger(logFn LogFunc) Middleware {
	if logFn == nil {
		logFn = func(LogEntry) {}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			logFn(LogEntry{
				Method:     r.Method,
				Path:       r.URL.Path,
				Status:     rec.status,
				Duration:   time.Since(start),
				BytesOut:   rec.bytes,
				RequestID:  RequestIDFromContext(r.Context()),
				RemoteAddr: r.RemoteAddr,
			})
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

// ─── Timeout ─────────────────────────────────────────────────────────────────

// Timeout wraps each request in a context with the given deadline.
// If the handler doesn't finish in time it returns 504. Note that the
// handler must honor ctx cancellation for Timeout to have any effect.
func Timeout(d time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, d, "request timeout")
	}
}

// ─── CORS ────────────────────────────────────────────────────────────────────

// CORSConfig configures the CORS middleware.
type CORSConfig struct {
	// AllowOrigins; use []string{"*"} for everywhere.
	AllowOrigins []string
	// AllowMethods defaults to GET, POST, PUT, PATCH, DELETE, OPTIONS.
	AllowMethods []string
	// AllowHeaders defaults to Content-Type, Authorization.
	AllowHeaders []string
	// AllowCredentials toggles Access-Control-Allow-Credentials.
	AllowCredentials bool
	// MaxAge seconds for preflight caching (default 600).
	MaxAge int
}

// CORS returns a minimal CORS middleware. For complex policies (regex
// origin match, per-route rules) use a dedicated CORS library.
func CORS(cfg CORSConfig) Middleware {
	if len(cfg.AllowMethods) == 0 {
		cfg.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	if len(cfg.AllowHeaders) == 0 {
		cfg.AllowHeaders = []string{"Content-Type", "Authorization"}
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 600
	}
	allowedOrigins := make(map[string]bool, len(cfg.AllowOrigins))
	wildcard := false
	for _, o := range cfg.AllowOrigins {
		if o == "*" {
			wildcard = true
			continue
		}
		allowedOrigins[o] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && (wildcard || allowedOrigins[origin]) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				if cfg.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowHeaders, ", "))
				w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", cfg.MaxAge))
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
