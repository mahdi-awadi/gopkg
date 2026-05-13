package llm

import (
	"context"
	"errors"
	"testing"
)

type stubProvider struct {
	name string
	resp ChatResponse
	err  error
}

func (s *stubProvider) Name() string { return s.name }
func (s *stubProvider) Chat(_ context.Context, _ ChatRequest) (ChatResponse, error) {
	return s.resp, s.err
}

func TestRegistryNoProvidersReturnsErr(t *testing.T) {
	r := NewRegistry()
	if _, err := r.Primary(); !errors.Is(err, ErrNoProviders) {
		t.Fatalf("Primary() err = %v, want ErrNoProviders", err)
	}
	if _, err := r.Chat(context.Background(), ChatRequest{}); !errors.Is(err, ErrNoProviders) {
		t.Fatalf("Chat() err = %v, want ErrNoProviders", err)
	}
}

func TestRegistryPrimaryReturnsFirstRegistered(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubProvider{name: "p1"})
	r.Register(&stubProvider{name: "p2"})
	got, err := r.Primary()
	if err != nil {
		t.Fatalf("Primary() err = %v", err)
	}
	if got.Name() != "p1" {
		t.Fatalf("Primary().Name() = %q, want p1", got.Name())
	}
}

func TestRegistryChatDelegatesToPrimary(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubProvider{name: "p1", resp: ChatResponse{Text: "hello", FinishReason: "stop"}})
	got, err := r.Chat(context.Background(), ChatRequest{UserText: "hi"})
	if err != nil {
		t.Fatalf("Chat() err = %v", err)
	}
	if got.Text != "hello" || got.FinishReason != "stop" {
		t.Fatalf("Chat() = %+v", got)
	}
}

func TestRegistryChatPropagatesErr(t *testing.T) {
	r := NewRegistry()
	want := errors.New("boom")
	r.Register(&stubProvider{name: "p1", err: want})
	if _, err := r.Chat(context.Background(), ChatRequest{}); !errors.Is(err, want) {
		t.Fatalf("Chat() err = %v, want %v", err, want)
	}
}
