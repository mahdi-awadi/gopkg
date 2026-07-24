package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// WhatsApp Business Calling — call-control only.
//
// This provider implements the REST call-control surface of Meta's WhatsApp
// Business Calling API (POST /{phone_number_id}/calls). It does NOT run a media
// stack: SDP offer/answer are treated as opaque strings supplied by the caller,
// and no RTP/audio is negotiated or carried here. A media/WebRTC pipeline is a
// separate concern and out of scope for this backend provider.

// CallAction is a WhatsApp Business Calling control action.
type CallAction string

const (
	// CallActionConnect initiates an outbound call (business → user). Requires an
	// SDP offer.
	CallActionConnect CallAction = "connect"
	// CallActionPreAccept pre-accepts an inbound call to speed up media setup.
	// Requires an SDP answer.
	CallActionPreAccept CallAction = "pre_accept"
	// CallActionAccept accepts an inbound call. Requires an SDP answer.
	CallActionAccept CallAction = "accept"
	// CallActionReject rejects an inbound call.
	CallActionReject CallAction = "reject"
	// CallActionTerminate ends an ongoing or connecting call.
	CallActionTerminate CallAction = "terminate"
)

// CallRequest describes a call-control action against the WhatsApp Business
// Calling API. SDPType/SDP are opaque passthrough values — this package never
// generates or parses SDP.
type CallRequest struct {
	// To is the recipient phone number (E.164). Required for Connect.
	To string
	// CallID identifies an existing call. Required for PreAccept/Accept/Reject/
	// Terminate; ignored for Connect.
	CallID string
	// SDPType is "offer" (Connect) or "answer" (PreAccept/Accept). Opaque.
	SDPType string
	// SDP is the opaque session description supplied by the caller's media stack.
	SDP string
	// CallbackData is echoed back on webhooks (biz_opaque_callback_data). Optional.
	CallbackData string
}

// CallsResponse is the envelope returned by /calls.
type CallsResponse struct {
	MessagingProduct string `json:"messaging_product,omitempty"`
	Calls            []struct {
		ID string `json:"id"`
	} `json:"calls,omitempty"`
	Success bool      `json:"success,omitempty"`
	Error   *APIError `json:"error,omitempty"`
}

// CallResult is the normalized result of a call-control action.
type CallResult struct {
	Success      bool
	ProviderCode string
	CallID       string
	Error        string
	RawResponse  []byte
}

// Connect initiates an outbound call to req.To with the caller-supplied SDP offer.
func (p *Provider) Connect(ctx context.Context, req *CallRequest) (*CallResult, error) {
	if req.To == "" {
		return callFailure("recipient phone is required"), nil
	}
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"to":                stripPhonePrefix(req.To),
		"action":            string(CallActionConnect),
		"session":           sessionObj("offer", req),
	}
	if req.CallbackData != "" {
		payload["biz_opaque_callback_data"] = req.CallbackData
	}
	return p.postCalls(ctx, payload)
}

// PreAccept pre-accepts an inbound call with the caller-supplied SDP answer.
func (p *Provider) PreAccept(ctx context.Context, req *CallRequest) (*CallResult, error) {
	return p.callAction(ctx, CallActionPreAccept, req, true)
}

// Accept accepts an inbound call with the caller-supplied SDP answer.
func (p *Provider) Accept(ctx context.Context, req *CallRequest) (*CallResult, error) {
	return p.callAction(ctx, CallActionAccept, req, true)
}

// Reject rejects an inbound call identified by req.CallID.
func (p *Provider) Reject(ctx context.Context, req *CallRequest) (*CallResult, error) {
	return p.callAction(ctx, CallActionReject, req, false)
}

// Terminate ends the call identified by req.CallID.
func (p *Provider) Terminate(ctx context.Context, req *CallRequest) (*CallResult, error) {
	return p.callAction(ctx, CallActionTerminate, req, false)
}

// RequestCallPermission asks a user for permission to be called, via an
// interactive message on the /messages endpoint.
func (p *Provider) RequestCallPermission(ctx context.Context, to string) (*provider_SendResult, error) {
	if to == "" {
		return &provider_SendResult{Success: false, Error: "recipient phone is required"}, nil
	}
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                stripPhonePrefix(to),
		"type":              "interactive",
		"interactive": map[string]any{
			"type": "call_permission_request",
		},
	}
	raw, ok, errMsg := p.doJSON(ctx, "messages", payload)
	return &provider_SendResult{Success: ok, Error: errMsg, RawResponse: raw}, nil
}

// provider_SendResult is a minimal result for the permission-request call, kept
// local so calling.go does not need the communication/provider types.
type provider_SendResult struct {
	Success     bool
	Error       string
	RawResponse []byte
}

func (p *Provider) callAction(ctx context.Context, action CallAction, req *CallRequest, needsSDP bool) (*CallResult, error) {
	if req.CallID == "" {
		return callFailure("call_id is required"), nil
	}
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"call_id":           req.CallID,
		"action":            string(action),
	}
	if needsSDP {
		payload["session"] = sessionObj("answer", req)
	}
	return p.postCalls(ctx, payload)
}

func sessionObj(defaultType string, req *CallRequest) map[string]any {
	t := req.SDPType
	if t == "" {
		t = defaultType
	}
	return map[string]any{"sdp_type": t, "sdp": req.SDP}
}

func callFailure(msg string) *CallResult {
	return &CallResult{Success: false, ProviderCode: ProviderCode, Error: msg}
}

func (p *Provider) postCalls(ctx context.Context, payload map[string]any) (*CallResult, error) {
	raw, ok, errMsg := p.doJSON(ctx, "calls", payload)
	res := &CallResult{Success: ok, ProviderCode: ProviderCode, Error: errMsg, RawResponse: raw}
	if ok {
		var parsed CallsResponse
		if json.Unmarshal(raw, &parsed) == nil && len(parsed.Calls) > 0 {
			res.CallID = parsed.Calls[0].ID
		}
	}
	return res, nil
}

// doJSON POSTs a JSON payload to /{phone_number_id}/{endpoint} and returns the
// raw body, a success flag, and a formatted error message on failure. Transport
// errors are returned as (nil, false, msg) so callers can surface them without a
// hard error, matching postMessage's behavior.
func (p *Provider) doJSON(ctx context.Context, endpoint string, payload map[string]any) ([]byte, bool, string) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, false, fmt.Sprintf("marshal: %v", err)
	}
	url := fmt.Sprintf("%s/%s/%s", p.cfg.GraphAPIBase, p.cfg.PhoneNumberID, endpoint)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, false, fmt.Sprintf("build request: %v", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.AccessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		p.logger.Error("meta: http failed", map[string]any{"error": err.Error(), "endpoint": endpoint})
		return nil, false, err.Error()
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, fmt.Sprintf("read body: %v", err)
	}

	var envelope struct {
		Error *APIError `json:"error,omitempty"`
	}
	_ = json.Unmarshal(raw, &envelope)
	if resp.StatusCode >= 300 || envelope.Error != nil {
		errMsg := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(raw))
		if envelope.Error != nil {
			errMsg = fmt.Sprintf("%d %s (%s)", envelope.Error.Code, envelope.Error.Message, envelope.Error.Type)
		}
		p.logger.Warn("meta: non-success response", map[string]any{"status": resp.StatusCode, "endpoint": endpoint, "error": errMsg})
		return raw, false, errMsg
	}
	return raw, true, ""
}
