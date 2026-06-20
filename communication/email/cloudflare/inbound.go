package cloudflare

import (
	"encoding/json"
	"fmt"
)

// Inbound is a normalized email received via the Cloudflare Email Worker.
type Inbound struct {
	From      string `json:"from"`
	FromName  string `json:"fromName"`
	To        string `json:"to"`
	Subject   string `json:"subject"`
	Text      string `json:"text"`
	HTML      string `json:"html"`
	MessageID string `json:"messageId"`
	InReplyTo string `json:"inReplyTo"`
}

// ParseInbound decodes the Email Worker's JSON payload.
func ParseInbound(body []byte) (Inbound, error) {
	var in Inbound
	if err := json.Unmarshal(body, &in); err != nil {
		return Inbound{}, fmt.Errorf("cloudflare: parse inbound: %w", err)
	}
	if in.From == "" {
		return Inbound{}, fmt.Errorf("cloudflare: inbound missing From")
	}
	return in, nil
}
