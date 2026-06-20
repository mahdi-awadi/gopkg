package cloudflare

import "testing"

func TestParseInbound(t *testing.T) {
	body := []byte(`{"from":"cust@example.com","fromName":"Jane","to":"support@tech-gate.online",
		"subject":"Order status?","messageId":"<abc@mail>","inReplyTo":"","text":"where is my order","html":"<p>where is my order</p>"}`)
	in, err := ParseInbound(body)
	if err != nil {
		t.Fatalf("ParseInbound: %v", err)
	}
	if in.From != "cust@example.com" || in.Subject != "Order status?" || in.Text != "where is my order" || in.MessageID != "<abc@mail>" {
		t.Fatalf("parsed wrong: %+v", in)
	}
}

func TestParseInboundRejectsNoFrom(t *testing.T) {
	if _, err := ParseInbound([]byte(`{"subject":"x"}`)); err == nil {
		t.Fatal("expected error when From is missing")
	}
}
