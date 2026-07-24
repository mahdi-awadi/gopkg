package meta

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConnect_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/calls") {
			t.Errorf("expected /calls path, got %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		if !strings.Contains(s, `"action":"connect"`) {
			t.Errorf("expected action connect, got %s", s)
		}
		if !strings.Contains(s, `"sdp":"OFFER-SDP"`) || !strings.Contains(s, `"sdp_type":"offer"`) {
			t.Errorf("expected opaque offer sdp passthrough, got %s", s)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"messaging_product":"whatsapp","calls":[{"id":"wacid.123"}]}`))
	}))
	defer srv.Close()

	p := New(Config{PhoneNumberID: "p", AccessToken: "t", GraphAPIBase: srv.URL}, nil)
	res, err := p.Connect(context.Background(), &CallRequest{To: "+1555", SDP: "OFFER-SDP"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.Success || res.CallID != "wacid.123" {
		t.Fatalf("got %+v", res)
	}
}

func TestAccept_SendsAnswerSDPAndCallID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		if !strings.Contains(s, `"action":"accept"`) || !strings.Contains(s, `"call_id":"wacid.9"`) {
			t.Errorf("expected accept + call_id, got %s", s)
		}
		if !strings.Contains(s, `"sdp_type":"answer"`) {
			t.Errorf("expected answer sdp_type, got %s", s)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"messaging_product":"whatsapp","calls":[{"id":"wacid.9"}]}`))
	}))
	defer srv.Close()

	p := New(Config{PhoneNumberID: "p", AccessToken: "t", GraphAPIBase: srv.URL}, nil)
	res, _ := p.Accept(context.Background(), &CallRequest{CallID: "wacid.9", SDP: "ANSWER-SDP"})
	if !res.Success {
		t.Fatalf("expected success, got %+v", res)
	}
}

func TestTerminate_RequiresCallID(t *testing.T) {
	p := New(Config{PhoneNumberID: "p", AccessToken: "t"}, nil)
	res, _ := p.Terminate(context.Background(), &CallRequest{})
	if res.Success || !strings.Contains(res.Error, "call_id") {
		t.Fatalf("expected call_id error, got %+v", res)
	}
}

func TestReject_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": &APIError{Code: 138001, Message: "call not found", Type: "CallingError"},
		})
	}))
	defer srv.Close()

	p := New(Config{PhoneNumberID: "p", AccessToken: "t", GraphAPIBase: srv.URL}, nil)
	res, _ := p.Reject(context.Background(), &CallRequest{CallID: "wacid.x"})
	if res.Success || !strings.Contains(res.Error, "138001") {
		t.Fatalf("expected calling error 138001, got %+v", res)
	}
}

func TestRequestCallPermission_Interactive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/messages") {
			t.Errorf("expected /messages, got %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"call_permission_request"`) {
			t.Errorf("expected call_permission_request, got %s", string(body))
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"messaging_product":"whatsapp","messages":[{"id":"wamid.perm"}]}`))
	}))
	defer srv.Close()

	p := New(Config{PhoneNumberID: "p", AccessToken: "t", GraphAPIBase: srv.URL}, nil)
	res, _ := p.RequestCallPermission(context.Background(), "+1555")
	if !res.Success {
		t.Fatalf("expected success, got %+v", res)
	}
}

func TestConnect_RequiresRecipient(t *testing.T) {
	p := New(Config{PhoneNumberID: "p", AccessToken: "t"}, nil)
	res, _ := p.Connect(context.Background(), &CallRequest{})
	if res.Success || !strings.Contains(res.Error, "recipient") {
		t.Fatalf("expected recipient error, got %+v", res)
	}
}
