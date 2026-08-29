package adminli

import "encoding/json"

// Wire types for Adminli's Programmable Messaging API.
// Contract: /home/adminli/proto/adminli/messaging/v1/messaging.proto
// Published: https://developers.adminli.app/messaging.yaml
// JSON is camelCase because the OpenAPI document is generated from proto.

// SendMessageRequest is the POST /v1/messages body.
type SendMessageRequest struct {
	To       string        `json:"to"`
	Template *TemplateSend `json:"template,omitempty"`
}

// TemplateSend delivers an approved WhatsApp template.
type TemplateSend struct {
	Name             string   `json:"name"`
	Language         string   `json:"language,omitempty"`
	BodyParameters   []string `json:"bodyParameters,omitempty"`
	ButtonParameters []string `json:"buttonParameters,omitempty"`
}

// SendMessageResponse is the POST /v1/messages 200 body.
type SendMessageResponse struct {
	MessageID string `json:"messageId"`
	To        string `json:"to"`
	Channel   string `json:"channel"`
	Status    string `json:"status"`
}

// APIError models the two error shapes observed live against api.adminli.app:
//
//   - auth layer (bad/missing API key), HTTP 401: {"error":"invalid_api_key"}
//   - RPC layer (Connect-style), any status:       {"code":13,"message":"internal"}
//     {"code":3,"message":"invalid_phone"}
//
// Both shapes may be present in the same body, so every field decodes
// independently rather than picking one shape.
type APIError struct {
	// Error is the auth-middleware shape: a bare string reason.
	Error string `json:"error"`
	// Code and Message are the RPC-layer (Connect) shape.
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Details json.RawMessage `json:"details,omitempty"`
}

// Reason returns the most specific token available: the RPC-layer Message when
// present, else the auth-layer Error, else empty (caller falls back to the raw body).
func (e APIError) Reason() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Error
}
