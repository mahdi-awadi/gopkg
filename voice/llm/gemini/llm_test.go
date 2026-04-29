package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mahdi-awadi/gopkg/voice/pipeline"
)

func dialMockGemini(t *testing.T, handler func(*websocket.Conn)) (*LLM, func()) {
	t.Helper()
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		handler(conn)
	}))
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		srv.Close()
		t.Fatalf("dial: %v", err)
	}
	llm := NewLLM(client, Options{})
	return llm, func() { _ = client.Close(); srv.Close() }
}

func TestLLM_ImplementsPipelineLLM(t *testing.T) {
	var _ pipeline.LLM = (*LLM)(nil)
}

func TestLLM_FormatsMatchGeminiLiveAudio(t *testing.T) {
	llm := &LLM{}
	if got := llm.InboundFormat(); got.Encoding != pipeline.EncodingPCM16LE || got.SampleRate != 24000 || got.Channels != 1 {
		t.Fatalf("InboundFormat=%+v, want pcm16le@24000 mono", got)
	}
	if got := llm.OutboundFormat(); got.Encoding != pipeline.EncodingPCM16LE || got.SampleRate != 16000 || got.Channels != 1 {
		t.Fatalf("OutboundFormat=%+v, want pcm16le@16000 mono", got)
	}
}

func TestBuildURLRequiresAPIKeyAndRedactsIt(t *testing.T) {
	_, err := BuildURL(Config{APIKey: ""})
	if err == nil {
		t.Fatal("expected missing API key error")
	}
	raw, err := BuildURL(Config{Endpoint: "wss://example.test/live?alt=json", APIKey: "secret-key"})
	if err != nil {
		t.Fatalf("BuildURL: %v", err)
	}
	if !strings.Contains(raw, "key=secret-key") || !strings.Contains(raw, "alt=json") {
		t.Fatalf("raw URL=%q", raw)
	}
	redacted := RedactURL(raw)
	if strings.Contains(redacted, "secret-key") || !strings.Contains(redacted, "key=REDACTED") {
		t.Fatalf("redacted URL leaked key: %q", redacted)
	}
}

func TestLLM_OpenSendsSetupHistoryAndGreeting(t *testing.T) {
	received := make(chan map[string]any, 5)
	llm, cleanup := dialMockGemini(t, func(conn *websocket.Conn) {
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var m map[string]any
			_ = json.Unmarshal(raw, &m)
			received <- m
			if _, ok := m["setup"]; ok {
				_ = conn.WriteJSON(ServerMessage{SetupComplete: &struct{}{}})
			}
		}
	})
	defer cleanup()

	setup := pipeline.SetupRequest{
		SystemPrompt: "You are useful.",
		Voice:        "Puck",
		Tools: []pipeline.ToolDecl{{
			Name:        "lookup_order",
			Description: "Look up an order",
			Parameters: pipeline.ToolSchema{
				Type: "object",
				Properties: map[string]pipeline.ToolProperty{
					"id": {Type: "string", Description: "Order ID"},
				},
				Required: []string{"id"},
			},
		}},
		History: []pipeline.HistoryTurn{
			{Role: pipeline.RoleUser, Content: "hi"},
			{Role: pipeline.RoleAssistant, Content: "hello"},
		},
	}
	if err := llm.Open(context.Background(), setup); err != nil {
		t.Fatalf("Open: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	close(received)

	var got []map[string]any
	for m := range received {
		got = append(got, m)
	}
	if len(got) != 4 {
		t.Fatalf("got %d messages, want setup + 2 history + greeting", len(got))
	}
	setupMsg := got[0]["setup"].(map[string]any)
	if setupMsg["model"] != DefaultModel {
		t.Fatalf("model=%v", setupMsg["model"])
	}
	if _, ok := setupMsg["inputAudioTranscription"]; ok {
		t.Fatalf("inputAudioTranscription should be opt-in, got %+v", setupMsg)
	}
	if _, ok := setupMsg["systemInstruction"]; !ok {
		t.Fatalf("systemInstruction missing: %+v", setupMsg)
	}
	if tools := setupMsg["tools"].([]any); len(tools) != 1 {
		t.Fatalf("tools=%v", tools)
	}
	if _, ok := got[1]["clientContent"]; !ok {
		t.Fatalf("history msg 1 not clientContent: %+v", got[1])
	}
	if _, ok := got[2]["clientContent"]; !ok {
		t.Fatalf("history msg 2 not clientContent: %+v", got[2])
	}
	if _, ok := got[3]["realtimeInput"]; !ok {
		t.Fatalf("greeting not realtimeInput: %+v", got[3])
	}
}

func TestLLM_OpenCanOptIntoInputAudioTranscription(t *testing.T) {
	received := make(chan map[string]any, 1)
	llm, cleanup := dialMockGemini(t, func(conn *websocket.Conn) {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var m map[string]any
		_ = json.Unmarshal(raw, &m)
		received <- m
		_ = conn.WriteJSON(ServerMessage{SetupComplete: &struct{}{}})
		_, _, _ = conn.ReadMessage()
	})
	defer cleanup()
	llm.opts.EnableInputTranscription = true

	if err := llm.Open(context.Background(), pipeline.SetupRequest{}); err != nil {
		t.Fatalf("Open: %v", err)
	}
	setupMsg := (<-received)["setup"].(map[string]any)
	if _, ok := setupMsg["inputAudioTranscription"]; !ok {
		t.Fatalf("inputAudioTranscription missing after opt-in: %+v", setupMsg)
	}
}

func TestLLM_SendAudioToolResultsAndInjectTurn(t *testing.T) {
	received := make(chan map[string]any, 4)
	llm, cleanup := dialMockGemini(t, func(conn *websocket.Conn) {
		greetingSeen := false
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var m map[string]any
			_ = json.Unmarshal(raw, &m)
			if _, ok := m["setup"]; ok {
				_ = conn.WriteJSON(ServerMessage{SetupComplete: &struct{}{}})
				continue
			}
			if _, ok := m["realtimeInput"]; ok && !greetingSeen {
				greetingSeen = true
				continue // greeting
			}
			received <- m
		}
	})
	defer cleanup()

	if err := llm.Open(context.Background(), pipeline.SetupRequest{}); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := llm.SendAudio(context.Background(), pipeline.Frame{Data: []byte{0x11, 0x22}}); err != nil {
		t.Fatalf("SendAudio: %v", err)
	}
	if err := llm.SendToolResults(context.Background(), []pipeline.ToolResult{{CallID: "c1", Data: map[string]any{"ok": true}}}); err != nil {
		t.Fatalf("SendToolResults: %v", err)
	}
	if err := llm.InjectTurn(context.Background(), pipeline.HistoryTurn{Role: pipeline.RoleUser, Content: "hint"}); err != nil {
		t.Fatalf("InjectTurn: %v", err)
	}

	audio := <-received
	audioData := audio["realtimeInput"].(map[string]any)["audio"].(map[string]any)
	if audioData["mimeType"] != "audio/pcm;rate=16000" || audioData["data"] != "ESI=" {
		t.Fatalf("audio=%+v", audioData)
	}
	tool := <-received
	if _, ok := tool["toolResponse"]; !ok {
		t.Fatalf("tool response missing: %+v", tool)
	}
	turn := <-received
	if _, ok := turn["clientContent"]; !ok {
		t.Fatalf("client content missing: %+v", turn)
	}
}

func TestLLM_EventsTranslateGeminiMessages(t *testing.T) {
	llm, cleanup := dialMockGemini(t, func(conn *websocket.Conn) {
		_, _, _ = conn.ReadMessage()
		_ = conn.WriteJSON(ServerMessage{SetupComplete: &struct{}{}})
		_, _, _ = conn.ReadMessage()
		_ = conn.WriteJSON(ServerMessage{ServerContent: &ServerContent{
			InputTranscription: &Transcription{Text: "caller"},
			ModelTurn: &Content{Parts: []Part{
				{InlineData: &InlineData{Data: "AAA="}},
				{Text: "reply"},
			}},
			Interrupted:  true,
			TurnComplete: true,
		}})
		_ = conn.WriteJSON(ServerMessage{ToolCall: &ToolCall{FunctionCalls: []FunctionCall{{ID: "c1", Name: "lookup", Args: map[string]any{"id": "A"}}}}})
		_ = conn.Close()
	})
	defer cleanup()

	if err := llm.Open(context.Background(), pipeline.SetupRequest{}); err != nil {
		t.Fatalf("Open: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	events, _ := llm.Events(ctx)

	var transcript, audio, partial, final, interrupted, complete, tool bool
	for ev := range events {
		switch e := ev.(type) {
		case pipeline.EventCallerTranscript:
			transcript = e.Text == "caller"
		case pipeline.EventAudioOut:
			audio = len(e.Frame.Data) > 0
		case pipeline.EventAssistantText:
			if e.Final {
				final = e.Text == "reply"
			} else {
				partial = e.Text == "reply"
			}
		case pipeline.EventInterrupted:
			interrupted = true
		case pipeline.EventTurnComplete:
			complete = true
		case pipeline.EventToolCalls:
			tool = len(e.Calls) == 1 && e.Calls[0].Name == "lookup"
		}
	}
	if !(transcript && audio && partial && final && interrupted && complete && tool) {
		t.Fatalf("missing events transcript=%v audio=%v partial=%v final=%v interrupted=%v complete=%v tool=%v",
			transcript, audio, partial, final, interrupted, complete, tool)
	}
}

func TestLLM_CloseIsIdempotent(t *testing.T) {
	llm, cleanup := dialMockGemini(t, func(conn *websocket.Conn) {
		_, _, _ = conn.ReadMessage()
	})
	defer cleanup()
	if err := llm.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := llm.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
