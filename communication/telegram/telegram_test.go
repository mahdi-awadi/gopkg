package telegram

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mahdi-awadi/gopkg/communication/provider"
)

func TestProvider_CodeAndChannels(t *testing.T) {
	p := New(Config{BotToken: "t"}, nil)
	if p.Code() != "telegram_bot" {
		t.Fatalf("code=%q", p.Code())
	}
	chs := p.SupportedChannels()
	if len(chs) != 1 || chs[0] != provider.ChannelTelegram {
		t.Fatalf("expected [telegram], got %v", chs)
	}
}

func TestValidateConfig(t *testing.T) {
	if err := New(Config{}, nil).ValidateConfig(); err == nil {
		t.Fatal("expected error when BotToken missing")
	}
	if err := New(Config{BotToken: "t"}, nil).ValidateConfig(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSend_NoChatIDFails(t *testing.T) {
	p := New(Config{BotToken: "t"}, nil)
	resp, _ := p.Send(context.Background(), &provider.SendRequest{Body: "hi"})
	if resp.Success {
		t.Fatal("expected failure without chat_id")
	}
}

func TestSend_HappyPathAgainstFakeServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		if !strings.Contains(r.URL.RawQuery, "chat_id=12345") {
			t.Errorf("expected chat_id=12345 in query, got %q", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":42}}`))
	}))
	defer srv.Close()

	p := New(Config{BotToken: "BOT:TOKEN", APIBaseURL: srv.URL}, nil)
	resp, err := p.Send(context.Background(), &provider.SendRequest{
		RecipientTelegramChatID: "12345",
		Body:                    "Hello",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got %+v", resp)
	}
}

func TestSend_APIErrorPropagates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":false,"error_code":403,"description":"Forbidden"}`))
	}))
	defer srv.Close()

	p := New(Config{BotToken: "t", APIBaseURL: srv.URL}, nil)
	resp, err := p.Send(context.Background(), &provider.SendRequest{
		RecipientTelegramChatID: "12345",
		Body:                    "Hi",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Success || !strings.Contains(resp.Error, "403") {
		t.Fatalf("expected 403 error, got %+v", resp)
	}
}
