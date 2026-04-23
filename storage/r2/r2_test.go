package r2

import (
	"strings"
	"testing"
	"time"
)

func TestNew_IncompleteConfigReturnsError(t *testing.T) {
	cases := []struct {
		name string
		cfg  *Config
	}{
		{name: "nil", cfg: nil},
		{name: "no access key", cfg: &Config{SecretKey: "s", Endpoint: "e", BucketName: "b"}},
		{name: "no secret key", cfg: &Config{AccessKey: "a", Endpoint: "e", BucketName: "b"}},
		{name: "no endpoint", cfg: &Config{AccessKey: "a", SecretKey: "s", BucketName: "b"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := New(c.cfg)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), "incomplete configuration") {
				t.Fatalf("expected 'incomplete configuration' error, got %q", err)
			}
		})
	}
}

func TestNew_CompleteConfigSucceeds(t *testing.T) {
	c, err := New(&Config{
		AccessKey:  "ak",
		SecretKey:  "sk",
		Endpoint:   "https://example.r2.cloudflarestorage.com",
		Subdomain:  "cdn.example.com",
		BucketName: "bucket",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil || c.client == nil {
		t.Fatalf("expected non-nil client")
	}
}

func TestGetSignedURL_ProducesNonEmpty(t *testing.T) {
	c, _ := New(&Config{
		AccessKey:  "ak",
		SecretKey:  "sk",
		Endpoint:   "https://example.r2.cloudflarestorage.com",
		BucketName: "bucket",
	})
	url, err := c.GetSignedURL("key", time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url == "" || !strings.Contains(url, "bucket/key") {
		t.Fatalf("expected signed URL containing bucket/key, got %q", url)
	}
}
