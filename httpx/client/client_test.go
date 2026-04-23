package client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNew_DefaultsApplied(t *testing.T) {
	c := New(Config{})
	if c.Timeout != 30*time.Second {
		t.Fatalf("default timeout, got %v", c.Timeout)
	}
	if c.Transport == nil {
		t.Fatal("transport should not be nil")
	}
}

func TestRetry_Successful_NoRetries(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(Config{
		Retry: RetryConfig{MaxAttempts: 3, BackoffInitial: time.Millisecond},
	})
	resp, err := c.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	if hits != 1 {
		t.Fatalf("expected 1 hit, got %d", hits)
	}
}

func TestRetry_On503_RetriesUntilOK(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(Config{
		Retry: RetryConfig{MaxAttempts: 5, BackoffInitial: time.Millisecond},
	})
	resp, err := c.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if hits != 3 {
		t.Fatalf("expected 3 hits, got %d", hits)
	}
}

func TestRetry_RespectsMaxAttempts(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	c := New(Config{
		Retry: RetryConfig{MaxAttempts: 3, BackoffInitial: time.Millisecond},
	})
	resp, err := c.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	if hits != 3 {
		t.Fatalf("expected 3 attempts, got %d", hits)
	}
}

func TestRetry_PreservesBodyAcrossRetries(t *testing.T) {
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		if len(bodies) < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(Config{
		Retry: RetryConfig{MaxAttempts: 3, BackoffInitial: time.Millisecond},
	})
	resp, err := c.Post(srv.URL, "text/plain", strings.NewReader("hello"))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	if len(bodies) != 2 {
		t.Fatalf("expected 2 hits, got %d", len(bodies))
	}
	if bodies[0] != "hello" || bodies[1] != "hello" {
		t.Fatalf("body not replayed: %v", bodies)
	}
}

func TestNew_CustomTransportUsed(t *testing.T) {
	called := false
	custom := roundTripFn(func(r *http.Request) (*http.Response, error) {
		called = true
		return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
	})
	c := New(Config{Transport: custom})
	resp, err := c.Get("http://example.com")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if !called {
		t.Fatal("custom transport not used")
	}
}

type roundTripFn func(r *http.Request) (*http.Response, error)

func (f roundTripFn) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
