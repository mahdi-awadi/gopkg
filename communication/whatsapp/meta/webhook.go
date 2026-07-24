package meta

import (
	"encoding/json"
	"fmt"
)

// Inbound webhook parsing for WhatsApp Cloud API.
//
// Meta delivers inbound messages, delivery statuses, and call events to a single
// webhook URL as an "whatsapp_business_account" object. ParseWebhook flattens the
// nested entry/changes/value structure into the events a consumer cares about.

// InboundMessage is a received WhatsApp message (text only is fully typed; other
// types expose Type + the raw JSON for the consumer).
type InboundMessage struct {
	ID          string          `json:"id"`
	From        string          `json:"from"`
	Timestamp   string          `json:"timestamp"`
	Type        string          `json:"type"`
	Text        string          `json:"-"`
	PhoneNumber string          `json:"-"` // business phone_number_id from metadata
	Raw         json.RawMessage `json:"-"`
}

// CallEvent is an inbound WhatsApp Business Calling webhook event.
type CallEvent struct {
	ID        string          `json:"id"`
	From      string          `json:"from"`
	To        string          `json:"to"`
	Event     string          `json:"event"`     // e.g. "connect", "terminate"
	Direction string          `json:"direction"` // "BUSINESS_INITIATED" / "USER_INITIATED"
	Status    string          `json:"status"`
	Timestamp string          `json:"timestamp"`
	Session   *CallSession    `json:"session,omitempty"` // SDP passthrough, opaque
	Raw       json.RawMessage `json:"-"`
}

// CallSession carries the opaque SDP for an inbound call event.
type CallSession struct {
	SDPType string `json:"sdp_type"`
	SDP     string `json:"sdp"`
}

// WebhookEvent is the flattened result of parsing one webhook POST body.
type WebhookEvent struct {
	Object   string
	Messages []InboundMessage
	Calls    []CallEvent
	// Statuses holds raw delivery-status objects for consumers that need them.
	Statuses []json.RawMessage
}

// rawWebhook mirrors the on-the-wire shape just enough to flatten it.
type rawWebhook struct {
	Object string `json:"object"`
	Entry  []struct {
		Changes []struct {
			Field string `json:"field"`
			Value struct {
				Metadata struct {
					PhoneNumberID string `json:"phone_number_id"`
				} `json:"metadata"`
				Messages []struct {
					ID        string `json:"id"`
					From      string `json:"from"`
					Timestamp string `json:"timestamp"`
					Type      string `json:"type"`
					Text      struct {
						Body string `json:"body"`
					} `json:"text"`
				} `json:"messages"`
				Calls    []json.RawMessage `json:"calls"`
				Statuses []json.RawMessage `json:"statuses"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

// ParseWebhook parses a webhook POST body into a flattened WebhookEvent.
func ParseWebhook(raw []byte) (*WebhookEvent, error) {
	var rw rawWebhook
	if err := json.Unmarshal(raw, &rw); err != nil {
		return nil, fmt.Errorf("meta: parse webhook: %w", err)
	}
	ev := &WebhookEvent{Object: rw.Object}
	for _, entry := range rw.Entry {
		for _, ch := range entry.Changes {
			pnID := ch.Value.Metadata.PhoneNumberID
			for _, m := range ch.Value.Messages {
				ev.Messages = append(ev.Messages, InboundMessage{
					ID:          m.ID,
					From:        m.From,
					Timestamp:   m.Timestamp,
					Type:        m.Type,
					Text:        m.Text.Body,
					PhoneNumber: pnID,
				})
			}
			for _, c := range ch.Value.Calls {
				var ce CallEvent
				if err := json.Unmarshal(c, &ce); err != nil {
					return nil, fmt.Errorf("meta: parse call event: %w", err)
				}
				ce.Raw = c
				ev.Calls = append(ev.Calls, ce)
			}
			ev.Statuses = append(ev.Statuses, ch.Value.Statuses...)
		}
	}
	return ev, nil
}

// VerifyTokenChallenge implements the GET webhook-verification handshake. When
// mode is "subscribe" and token matches expectedToken, it returns the challenge
// string to echo back and true; otherwise "", false.
func VerifyTokenChallenge(mode, token, challenge, expectedToken string) (string, bool) {
	if mode == "subscribe" && token != "" && token == expectedToken {
		return challenge, true
	}
	return "", false
}
