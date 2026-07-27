package meta

import (
	"encoding/json"
	"testing"
)

func TestParseWebhook_Message(t *testing.T) {
	raw := []byte(`{
	  "object":"whatsapp_business_account",
	  "entry":[{"changes":[{"field":"messages","value":{
	    "metadata":{"phone_number_id":"PNID1"},
	    "messages":[{"id":"wamid.1","from":"15551234","timestamp":"170","type":"text","text":{"body":"hello"}}]
	  }}]}]
	}`)
	ev, err := ParseWebhook(raw)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if ev.Object != "whatsapp_business_account" {
		t.Fatalf("object=%q", ev.Object)
	}
	if len(ev.Messages) != 1 {
		t.Fatalf("want 1 message, got %d", len(ev.Messages))
	}
	m := ev.Messages[0]
	if m.ID != "wamid.1" || m.From != "15551234" || m.Text != "hello" || m.PhoneNumber != "PNID1" {
		t.Fatalf("bad message: %+v", m)
	}
}

func TestParseWebhook_CallEvent(t *testing.T) {
	raw := []byte(`{
	  "object":"whatsapp_business_account",
	  "entry":[{"changes":[{"field":"calls","value":{
	    "metadata":{"phone_number_id":"PNID1"},
	    "calls":[{"id":"wacid.7","from":"15551234","to":"15559999","event":"connect","direction":"USER_INITIATED","session":{"sdp_type":"offer","sdp":"X"}}]
	  }}]}]
	}`)
	ev, err := ParseWebhook(raw)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(ev.Calls) != 1 {
		t.Fatalf("want 1 call, got %d", len(ev.Calls))
	}
	c := ev.Calls[0]
	if c.ID != "wacid.7" || c.Event != "connect" || c.Session == nil || c.Session.SDP != "X" {
		t.Fatalf("bad call event: %+v (session=%+v)", c, c.Session)
	}
}

func TestParseWebhook_OrderRaw(t *testing.T) {
	raw := []byte(`{
	  "object":"whatsapp_business_account",
	  "entry":[{"changes":[{"field":"messages","value":{
	    "metadata":{"phone_number_id":"PNID1"},
	    "messages":[{"id":"wamid.9","from":"15551234","timestamp":"171","type":"order","order":{
	      "catalog_id":"CAT1",
	      "product_items":[{"product_retailer_id":"SKU-1","quantity":2,"item_price":9.5,"currency":"USD"}]
	    }}]
	  }}]}]
	}`)
	ev, err := ParseWebhook(raw)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(ev.Messages) != 1 {
		t.Fatalf("want 1 message, got %d", len(ev.Messages))
	}
	m := ev.Messages[0]
	if m.Type != "order" {
		t.Fatalf("Type=%q, want order", m.Type)
	}
	if len(m.Raw) == 0 {
		t.Fatal("Raw was not populated for the order message")
	}
	var parsed struct {
		Order struct {
			ProductItems []struct {
				ProductRetailerID string `json:"product_retailer_id"`
			} `json:"product_items"`
		} `json:"order"`
	}
	if err := json.Unmarshal(m.Raw, &parsed); err != nil {
		t.Fatalf("unmarshal Raw: %v", err)
	}
	if len(parsed.Order.ProductItems) != 1 || parsed.Order.ProductItems[0].ProductRetailerID != "SKU-1" {
		t.Fatalf("product_retailer_id not recoverable from Raw: %+v", parsed)
	}
}

func TestParseWebhook_Malformed(t *testing.T) {
	if _, err := ParseWebhook([]byte(`{not json`)); err == nil {
		t.Fatal("expected error on malformed body")
	}
}

func TestVerifyTokenChallenge(t *testing.T) {
	if ch, ok := VerifyTokenChallenge("subscribe", "secret", "CH", "secret"); !ok || ch != "CH" {
		t.Fatalf("expected challenge echo, got %q ok=%v", ch, ok)
	}
	if _, ok := VerifyTokenChallenge("subscribe", "wrong", "CH", "secret"); ok {
		t.Fatal("expected failure on token mismatch")
	}
	if _, ok := VerifyTokenChallenge("unsubscribe", "secret", "CH", "secret"); ok {
		t.Fatal("expected failure on non-subscribe mode")
	}
}
