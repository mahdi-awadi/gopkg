package provider

import (
	"context"
	"errors"
	"testing"
)

type fakeProvider struct {
	code     string
	channels []Channel
	enabled  bool
	sent     int
}

func (f *fakeProvider) Code() string                 { return f.code }
func (f *fakeProvider) SupportedChannels() []Channel { return f.channels }
func (f *fakeProvider) Send(_ context.Context, _ *SendRequest) (*SendResponse, error) {
	f.sent++
	return &SendResponse{Success: true, ProviderCode: f.code}, nil
}
func (f *fakeProvider) GetStatus(_ context.Context, id string) (*DeliveryStatus, error) {
	return &DeliveryStatus{MessageID: id, Status: StatusDelivered}, nil
}
func (f *fakeProvider) ValidateConfig() error { return nil }
func (f *fakeProvider) Enabled() bool         { return f.enabled }

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	p := &fakeProvider{code: "fake", channels: []Channel{ChannelSMS}, enabled: true}
	if err := r.Register(p); err != nil {
		t.Fatalf("register: %v", err)
	}
	if r.Len() != 1 {
		t.Fatalf("expected len 1, got %d", r.Len())
	}
	got := r.Get("fake")
	if got == nil {
		t.Fatalf("Get returned nil")
	}
	if got.Code() != "fake" {
		t.Fatalf("expected fake, got %q", got.Code())
	}
}

func TestRegistry_RegisterDuplicateFails(t *testing.T) {
	r := NewRegistry()
	p1 := &fakeProvider{code: "x", channels: []Channel{ChannelSMS}}
	p2 := &fakeProvider{code: "x", channels: []Channel{ChannelEmail}}
	if err := r.Register(p1); err != nil {
		t.Fatal(err)
	}
	err := r.Register(p2)
	if err == nil {
		t.Fatal("expected duplicate error, got nil")
	}
}

func TestRegistry_RegisterNilOrEmptyCodeFails(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(nil); err == nil {
		t.Fatal("expected error for nil provider")
	}
	if err := r.Register(&fakeProvider{}); err == nil {
		t.Fatal("expected error for empty code")
	}
}

func TestRegistry_ByChannel(t *testing.T) {
	r := NewRegistry()
	_ = r.Register(&fakeProvider{code: "a", channels: []Channel{ChannelSMS}})
	_ = r.Register(&fakeProvider{code: "b", channels: []Channel{ChannelSMS, ChannelEmail}})
	_ = r.Register(&fakeProvider{code: "c", channels: []Channel{ChannelEmail}})

	sms := r.ByChannel(ChannelSMS)
	if len(sms) != 2 {
		t.Fatalf("expected 2 SMS providers, got %d", len(sms))
	}
	email := r.ByChannel(ChannelEmail)
	if len(email) != 2 {
		t.Fatalf("expected 2 email providers, got %d", len(email))
	}
	push := r.ByChannel(ChannelPush)
	if len(push) != 0 {
		t.Fatalf("expected 0 push providers, got %d", len(push))
	}
}

func TestProviderError_ErrorAndUnwrap(t *testing.T) {
	inner := errors.New("rate limited")
	pe := NewProviderError("twilio", "20429", "too many requests", true, inner)
	msg := pe.Error()
	if msg == "" {
		t.Fatalf("empty Error()")
	}
	if !errors.Is(pe, inner) {
		t.Fatalf("errors.Is should find wrapped error")
	}
}
