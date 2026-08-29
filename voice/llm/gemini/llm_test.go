package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mahdi-awadi/gopkg/voice/pipeline"
)

// Domain-neutral fixtures used across tests. The package itself doesn't
// know — and shouldn't know — about Iraqi Arabic eticket prompts or
// Saffar Kuwaiti prompts; tests assert on values they set themselves.
const (
	testSystemPrompt = "You are a helpful test assistant."
	testWakeSignal   = "[CALL_CONNECTED]"
)

func testTools() []pipeline.ToolDecl {
	return []pipeline.ToolDecl{
		{
			Name:        "get_user_bookings",
			Description: "Return the caller's recent bookings.",
			Parameters: pipeline.ToolSchema{
				Type: "object",
				Properties: map[string]pipeline.ToolProperty{
					"limit": {Type: "integer", Description: "Max bookings to return."},
				},
				Required: []string{"limit"},
			},
		},
	}
}

// dialMockGemini builds an httptest WS server that plays the role of
// Gemini Live. The handler function receives the server-side *Conn
// and drives the exchange. The returned *LLM is connected to it.
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
	llm := NewLLM(client)
	return llm, func() { client.Close(); srv.Close() }
}

// setupReq builds a SetupRequest with the test system prompt + tool list
// + wake signal. Tests can override individual fields by mutating the
// returned value before passing it to Open.
func setupReq() pipeline.SetupRequest {
	return pipeline.SetupRequest{
		SystemPrompt: testSystemPrompt,
		Tools:        testTools(),
		Extra:        map[string]any{"wake_signal": testWakeSignal},
	}
}

func TestLLM_ImplementsPipelineLLM(t *testing.T) {
	var _ pipeline.LLM = (*LLM)(nil)
}

// Regression guard. gopkg's Transport doc says "Inbound = format of frames
// entering our pipeline from this endpoint". For LLM that means the audio
// Gemini EMITS (pcm16le@24k). Swapping these once (pre-fix on 2026-04-24)
// caused pipeline.Run to fail every real call with ErrFormatBridge:
// "mulaw@8000 → pcm16le@24000". The combined bridge pair (inbound caller
// audio → LLM input; LLM output → outbound caller audio) must resolve to
// the two codec-bridge entries the gopkg pipeline supports.
func TestLLM_Formats_MatchGopkgConvention(t *testing.T) {
	llm := &LLM{}
	in := llm.InboundFormat()
	if in.Encoding != pipeline.EncodingPCM16LE || in.SampleRate != 24000 || in.Channels != 1 {
		t.Errorf("InboundFormat=%+v, want pcm16le@24000 mono (audio LLM emits to us)", in)
	}
	out := llm.OutboundFormat()
	if out.Encoding != pipeline.EncodingPCM16LE || out.SampleRate != 16000 || out.Channels != 1 {
		t.Errorf("OutboundFormat=%+v, want pcm16le@16000 mono (audio we send to LLM)", out)
	}
}

func TestLLM_Open_SendsSetupAndReadsSetupComplete(t *testing.T) {
	var gotSetup GeminiSetup
	sawWake := make(chan struct{}, 1)

	llm, cleanup := dialMockGemini(t, func(conn *websocket.Conn) {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		_ = json.Unmarshal(raw, &gotSetup)
		_ = conn.WriteJSON(GeminiServerMessage{SetupComplete: &struct{}{}})
		_, _, err = conn.ReadMessage()
		if err == nil {
			sawWake <- struct{}{}
		}
	})
	defer cleanup()

	err := llm.Open(context.Background(), setupReq())
	if err != nil {
		t.Fatalf("Open err: %v", err)
	}

	if gotSetup.Setup.Model != LiveModel {
		t.Errorf("Model=%q", gotSetup.Setup.Model)
	}
	if gotSetup.Setup.GenerationConfig.SpeechConfig.VoiceConfig.PrebuiltVoiceConfig.VoiceName != VoiceName {
		t.Errorf("VoiceName=%q", gotSetup.Setup.GenerationConfig.SpeechConfig.VoiceConfig.PrebuiltVoiceConfig.VoiceName)
	}
	if gotSetup.Setup.InputAudioTranscription == nil {
		t.Errorf("InputAudioTranscription missing (must be at setup level, not generationConfig)")
	}
	if gotSetup.Setup.OutputAudioTranscription == nil {
		t.Errorf("OutputAudioTranscription missing (must be at setup level, not generationConfig)")
	}
	if gotSetup.Setup.SystemInstruction == nil ||
		len(gotSetup.Setup.SystemInstruction.Parts) == 0 ||
		gotSetup.Setup.SystemInstruction.Parts[0].Text != testSystemPrompt {
		t.Errorf("SystemInstruction = %+v, want text=%q", gotSetup.Setup.SystemInstruction, testSystemPrompt)
	}
	if len(gotSetup.Setup.Tools) == 0 {
		t.Errorf("expected tool decls when SetupRequest.Tools is non-empty, got none")
	} else {
		decls := gotSetup.Setup.Tools[0].FunctionDeclarations
		if len(decls) != 1 || decls[0].Name != "get_user_bookings" {
			t.Errorf("functionDeclarations=%+v, want one decl named get_user_bookings", decls)
		}
	}
	select {
	case <-sawWake:
	case <-time.After(500 * time.Millisecond):
		t.Error("wake signal never sent")
	}
}

// When SetupRequest.Tools is empty, the wire payload must omit "tools"
// entirely. Replaces the old Registered=false test which exercised the
// same intent through the now-removed Options struct.
func TestLLM_Open_NoToolsWhenSetupTools_Empty(t *testing.T) {
	var gotSetup GeminiSetup
	llm, cleanup := dialMockGemini(t, func(conn *websocket.Conn) {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		_ = json.Unmarshal(raw, &gotSetup)
		_ = conn.WriteJSON(GeminiServerMessage{SetupComplete: &struct{}{}})
		_, _, _ = conn.ReadMessage() // drain whatever (or close)
	})
	defer cleanup()

	req := pipeline.SetupRequest{SystemPrompt: testSystemPrompt} // no Tools, no wake signal
	if err := llm.Open(context.Background(), req); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if len(gotSetup.Setup.Tools) != 0 {
		t.Errorf("Tools=%+v, want empty when SetupRequest.Tools is nil", gotSetup.Setup.Tools)
	}
}

// When no wake signal is supplied, Open must not send any extra message
// after history replay.
func TestLLM_Open_NoWakeSignalWhenHistoryPresent(t *testing.T) {
	received := make(chan any, 5)
	llm, cleanup := dialMockGemini(t, func(conn *websocket.Conn) {
		_, _, _ = conn.ReadMessage() // setup
		_ = conn.WriteJSON(GeminiServerMessage{SetupComplete: &struct{}{}})
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var m map[string]any
			_ = json.Unmarshal(raw, &m)
			received <- m
		}
	})
	defer cleanup()

	req := pipeline.SetupRequest{
		SystemPrompt: testSystemPrompt,
		History: []pipeline.HistoryTurn{
			{Role: pipeline.RoleUser, Content: "hello"},
		},
		// Extra omitted → no wake signal.
	}
	if err := llm.Open(context.Background(), req); err != nil {
		t.Fatalf("Open err: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	close(received)
	var got []map[string]any
	for m := range received {
		got = append(got, m.(map[string]any))
	}
	if len(got) != 1 {
		t.Fatalf("got %d messages, want 1 (history only, no wake signal)", len(got))
	}
	if _, ok := got[0]["clientContent"]; !ok {
		t.Errorf("msg[0] not clientContent: %+v", got[0])
	}
}

func TestLLM_Open_ReplaysHistoryBeforeWakeSignal(t *testing.T) {
	received := make(chan any, 5)
	llm, cleanup := dialMockGemini(t, func(conn *websocket.Conn) {
		_, _, _ = conn.ReadMessage() // setup
		_ = conn.WriteJSON(GeminiServerMessage{SetupComplete: &struct{}{}})
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var m map[string]any
			_ = json.Unmarshal(raw, &m)
			received <- m
		}
	})
	defer cleanup()

	req := setupReq()
	req.History = []pipeline.HistoryTurn{
		{Role: pipeline.RoleUser, Content: "first user"},
		{Role: pipeline.RoleAssistant, Content: "first reply"},
	}
	err := llm.Open(context.Background(), req)
	if err != nil {
		t.Fatalf("Open err: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	close(received)
	var got []map[string]any
	for m := range received {
		got = append(got, m.(map[string]any))
	}
	if len(got) != 3 {
		t.Fatalf("got %d messages, want 3 (2 history + 1 wake signal)", len(got))
	}
	if _, ok := got[0]["clientContent"]; !ok {
		t.Errorf("msg[0] not clientContent: %+v", got[0])
	}
	if _, ok := got[1]["clientContent"]; !ok {
		t.Errorf("msg[1] not clientContent: %+v", got[1])
	}
	cc, ok := got[2]["clientContent"].(map[string]any)
	if !ok {
		t.Fatalf("msg[2] not clientContent (wake signal): %+v", got[2])
	}
	turns, _ := cc["turns"].([]any)
	if len(turns) != 1 {
		t.Fatalf("wake signal turns=%d, want 1", len(turns))
	}
	turn := turns[0].(map[string]any)
	if turn["role"] != "user" {
		t.Errorf("wake signal role=%v, want user", turn["role"])
	}
	parts := turn["parts"].([]any)
	if parts[0].(map[string]any)["text"] != testWakeSignal {
		t.Errorf("wake signal text=%v, want %q", parts[0].(map[string]any)["text"], testWakeSignal)
	}
}

func TestLLM_Open_ReturnsErrorWhenSetupCompleteMissing(t *testing.T) {
	llm, cleanup := dialMockGemini(t, func(conn *websocket.Conn) {
		_, _, _ = conn.ReadMessage() // setup
		_ = conn.WriteJSON(GeminiServerMessage{})
	})
	defer cleanup()
	err := llm.Open(context.Background(), setupReq())
	if err == nil {
		t.Error("expected error when setupComplete missing")
	}
}

var _ = errors.New
var _ = sync.Mutex{}

func TestLLM_SendAudio_EmitsRealtimeInput(t *testing.T) {
	var gotRaw []byte
	llm, cleanup := dialMockGemini(t, func(conn *websocket.Conn) {
		_, _, _ = conn.ReadMessage() // setup
		_ = conn.WriteJSON(GeminiServerMessage{SetupComplete: &struct{}{}})
		_, _, _ = conn.ReadMessage() // wake signal
		_, gotRaw, _ = conn.ReadMessage()
	})
	defer cleanup()

	if err := llm.Open(context.Background(), setupReq()); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := llm.SendAudio(context.Background(), pipeline.Frame{Data: []byte{0x11, 0x22}}); err != nil {
		t.Fatalf("SendAudio: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	var m map[string]any
	_ = json.Unmarshal(gotRaw, &m)
	ri := m["realtimeInput"].(map[string]any)
	audio := ri["audio"].(map[string]any)
	if audio["mimeType"] != "audio/pcm;rate=16000" {
		t.Errorf("mimeType=%v", audio["mimeType"])
	}
	// Data is base64("\x11\x22") = "ESI="
	if audio["data"] != "ESI=" {
		t.Errorf("data=%v, want ESI=", audio["data"])
	}
}

func TestLLM_SendToolResults_EmitsToolResponse(t *testing.T) {
	var gotRaw []byte
	llm, cleanup := dialMockGemini(t, func(conn *websocket.Conn) {
		_, _, _ = conn.ReadMessage() // setup
		_ = conn.WriteJSON(GeminiServerMessage{SetupComplete: &struct{}{}})
		_, _, _ = conn.ReadMessage() // wake signal
		_, gotRaw, _ = conn.ReadMessage()
	})
	defer cleanup()

	if err := llm.Open(context.Background(), setupReq()); err != nil {
		t.Fatalf("Open: %v", err)
	}
	results := []pipeline.ToolResult{
		{CallID: "c1", Data: map[string]any{"ok": true}},
		{CallID: "c2", Data: map[string]any{"count": 3.0}},
	}
	if err := llm.SendToolResults(context.Background(), results); err != nil {
		t.Fatalf("SendToolResults: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	var m map[string]any
	_ = json.Unmarshal(gotRaw, &m)
	resps := m["toolResponse"].(map[string]any)["functionResponses"].([]any)
	if len(resps) != 2 {
		t.Fatalf("functionResponses=%d, want 2", len(resps))
	}
	r0 := resps[0].(map[string]any)
	if r0["id"] != "c1" {
		t.Errorf("resp[0].id=%v", r0["id"])
	}
}

func TestLLM_InjectTurn_EmitsClientContent(t *testing.T) {
	var gotRaw []byte
	llm, cleanup := dialMockGemini(t, func(conn *websocket.Conn) {
		_, _, _ = conn.ReadMessage() // setup
		_ = conn.WriteJSON(GeminiServerMessage{SetupComplete: &struct{}{}})
		_, _, _ = conn.ReadMessage() // wake signal
		_, gotRaw, _ = conn.ReadMessage()
	})
	defer cleanup()

	if err := llm.Open(context.Background(), setupReq()); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := llm.InjectTurn(context.Background(), pipeline.HistoryTurn{
		Role: pipeline.RoleUser, Content: "mid-session hint",
	}); err != nil {
		t.Fatalf("InjectTurn: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	var m map[string]any
	_ = json.Unmarshal(gotRaw, &m)
	cc := m["clientContent"].(map[string]any)
	turns := cc["turns"].([]any)
	t0 := turns[0].(map[string]any)
	if t0["role"] != "user" {
		t.Errorf("role=%v", t0["role"])
	}
	parts := t0["parts"].([]any)
	if parts[0].(map[string]any)["text"] != "mid-session hint" {
		t.Errorf("text=%v", parts[0].(map[string]any)["text"])
	}
}

func TestLLM_Close_IsIdempotent(t *testing.T) {
	llm, cleanup := dialMockGemini(t, func(conn *websocket.Conn) { <-make(chan struct{}) })
	defer cleanup()
	if err := llm.Close(); err != nil {
		t.Errorf("first Close: %v", err)
	}
	if err := llm.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}

// drainEvents reads all LLM events until channel closes or timeout.
func drainEvents(t *testing.T, llm *LLM, budget time.Duration) []pipeline.LLMEvent {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()
	ch, _ := llm.Events(ctx)
	var got []pipeline.LLMEvent
	for ev := range ch {
		got = append(got, ev)
	}
	return got
}

func TestLLM_Events_TranslatesCallerTranscript(t *testing.T) {
	llm, cleanup := dialMockGemini(t, func(conn *websocket.Conn) {
		_, _, _ = conn.ReadMessage()
		_ = conn.WriteJSON(GeminiServerMessage{SetupComplete: &struct{}{}})
		_, _, _ = conn.ReadMessage()
		_ = conn.WriteJSON(GeminiServerMessage{
			ServerContent: &GeminiServerContent{
				InputTranscription: &GeminiTranscription{Text: "hello"},
			},
		})
		_ = conn.Close()
	})
	defer cleanup()
	if err := llm.Open(context.Background(), setupReq()); err != nil {
		t.Fatal(err)
	}

	got := drainEvents(t, llm, 500*time.Millisecond)
	var sawTranscript bool
	for _, ev := range got {
		if ct, ok := ev.(pipeline.EventCallerTranscript); ok && ct.Text == "hello" {
			sawTranscript = true
		}
	}
	if !sawTranscript {
		t.Errorf("events=%+v, want EventCallerTranscript{hello}", got)
	}
}

func TestLLM_Events_TranslatesAudioOutAndText_AndFinalOnTurnComplete(t *testing.T) {
	llm, cleanup := dialMockGemini(t, func(conn *websocket.Conn) {
		_, _, _ = conn.ReadMessage()
		_ = conn.WriteJSON(GeminiServerMessage{SetupComplete: &struct{}{}})
		_, _, _ = conn.ReadMessage()
		_ = conn.WriteJSON(GeminiServerMessage{
			ServerContent: &GeminiServerContent{
				ModelTurn: &GeminiContent{
					Parts: []GeminiPart{
						{InlineData: &GeminiInlineData{Data: "AAA="}},
						{Text: "hi"},
					},
				},
			},
		})
		_ = conn.WriteJSON(GeminiServerMessage{
			ServerContent: &GeminiServerContent{TurnComplete: true},
		})
		_ = conn.Close()
	})
	defer cleanup()
	if err := llm.Open(context.Background(), setupReq()); err != nil {
		t.Fatal(err)
	}
	got := drainEvents(t, llm, 500*time.Millisecond)

	var audioOut bool
	var partialText, finalText bool
	var turnDone bool
	for _, ev := range got {
		switch e := ev.(type) {
		case pipeline.EventAudioOut:
			if len(e.Frame.Data) > 0 {
				audioOut = true
			}
		case pipeline.EventAssistantText:
			if e.Final {
				finalText = e.Text == "hi"
			} else {
				partialText = e.Text == "hi"
			}
		case pipeline.EventTurnComplete:
			turnDone = true
		}
	}
	if !audioOut {
		t.Errorf("no EventAudioOut: %+v", got)
	}
	if !partialText {
		t.Errorf("no streaming AssistantText: %+v", got)
	}
	if !finalText {
		t.Errorf("no final AssistantText on turnComplete: %+v", got)
	}
	if !turnDone {
		t.Errorf("no EventTurnComplete: %+v", got)
	}
}

func TestLLM_Events_TranslatesOutputTranscriptionToAssistantText(t *testing.T) {
	llm, cleanup := dialMockGemini(t, func(conn *websocket.Conn) {
		_, _, _ = conn.ReadMessage()
		_ = conn.WriteJSON(GeminiServerMessage{SetupComplete: &struct{}{}})
		_, _, _ = conn.ReadMessage()
		_ = conn.WriteJSON(GeminiServerMessage{
			ServerContent: &GeminiServerContent{
				OutputTranscription: &GeminiTranscription{Text: "hello from audio"},
			},
		})
		_ = conn.WriteJSON(GeminiServerMessage{
			ServerContent: &GeminiServerContent{TurnComplete: true},
		})
		_ = conn.Close()
	})
	defer cleanup()
	if err := llm.Open(context.Background(), setupReq()); err != nil {
		t.Fatal(err)
	}
	got := drainEvents(t, llm, 500*time.Millisecond)

	var finalText bool
	for _, ev := range got {
		if at, ok := ev.(pipeline.EventAssistantText); ok && at.Final && at.Text == "hello from audio" {
			finalText = true
		}
	}
	if !finalText {
		t.Errorf("events=%+v, want final EventAssistantText from outputTranscription", got)
	}
}

func TestLLM_Events_TranslatesInterruption(t *testing.T) {
	llm, cleanup := dialMockGemini(t, func(conn *websocket.Conn) {
		_, _, _ = conn.ReadMessage()
		_ = conn.WriteJSON(GeminiServerMessage{SetupComplete: &struct{}{}})
		_, _, _ = conn.ReadMessage()
		_ = conn.WriteJSON(GeminiServerMessage{
			ServerContent: &GeminiServerContent{Interrupted: true},
		})
		_ = conn.Close()
	})
	defer cleanup()
	if err := llm.Open(context.Background(), setupReq()); err != nil {
		t.Fatal(err)
	}
	got := drainEvents(t, llm, 500*time.Millisecond)
	var interrupted bool
	for _, ev := range got {
		if _, ok := ev.(pipeline.EventInterrupted); ok {
			interrupted = true
		}
	}
	if !interrupted {
		t.Errorf("no EventInterrupted: %+v", got)
	}
}

func TestLLM_Events_TranslatesToolCall(t *testing.T) {
	llm, cleanup := dialMockGemini(t, func(conn *websocket.Conn) {
		_, _, _ = conn.ReadMessage()
		_ = conn.WriteJSON(GeminiServerMessage{SetupComplete: &struct{}{}})
		_, _, _ = conn.ReadMessage()
		_ = conn.WriteJSON(GeminiServerMessage{
			ToolCall: &GeminiToolCall{
				FunctionCalls: []GeminiFunctionCall{
					{ID: "c1", Name: "get_user_bookings", Args: map[string]any{"limit": 5.0}},
				},
			},
		})
		_ = conn.Close()
	})
	defer cleanup()
	if err := llm.Open(context.Background(), setupReq()); err != nil {
		t.Fatal(err)
	}
	got := drainEvents(t, llm, 500*time.Millisecond)
	var toolCalls []pipeline.ToolCall
	for _, ev := range got {
		if tc, ok := ev.(pipeline.EventToolCalls); ok {
			toolCalls = tc.Calls
		}
	}
	if len(toolCalls) != 1 {
		t.Fatalf("got %d tool calls, want 1", len(toolCalls))
	}
	if toolCalls[0].ID != "c1" || toolCalls[0].Name != "get_user_bookings" {
		t.Errorf("call=%+v", toolCalls[0])
	}
}

// TestLLM_Open_SendsWakeSignalAsClientContent — the new contract:
// Extra["wake_signal"] is delivered as a clientContent user turn (not
// realtimeInput). Saffar relies on this so Gemini treats the wake
// payload as a system signal, not as user speech (which caused the
// 2026-05-13 role-inversion bug).
func TestLLM_Open_SendsWakeSignalAsClientContent(t *testing.T) {
	writes := make(chan []byte, 5)
	llm, cleanup := dialMockGemini(t, func(conn *websocket.Conn) {
		_, _, _ = conn.ReadMessage() // setup
		_ = conn.WriteJSON(GeminiServerMessage{SetupComplete: &struct{}{}})
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			writes <- raw
		}
	})
	defer cleanup()

	err := llm.Open(context.Background(), pipeline.SetupRequest{
		SystemPrompt: "you are a test agent",
		Tools:        nil,
		History:      nil,
		Extra: map[string]any{
			"wake_signal": "[CALL_CONNECTED]",
		},
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// Drain whatever the client sent after setup.
	time.Sleep(50 * time.Millisecond)
	close(writes)
	var got [][]byte
	for raw := range writes {
		got = append(got, raw)
	}
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 client write after setup (wake signal), got %d", len(got))
	}
	var first map[string]any
	if err := json.Unmarshal(got[0], &first); err != nil {
		t.Fatalf("unmarshal write: %v", err)
	}
	cc, ok := first["clientContent"].(map[string]any)
	if !ok {
		t.Fatalf("write missing clientContent: %s", got[0])
	}
	turns, _ := cc["turns"].([]any)
	if len(turns) != 1 {
		t.Fatalf("expected 1 turn, got %d", len(turns))
	}
	turn := turns[0].(map[string]any)
	if turn["role"] != "user" {
		t.Errorf("wake signal role = %q, want %q", turn["role"], "user")
	}
	parts := turn["parts"].([]any)
	if parts[0].(map[string]any)["text"] != "[CALL_CONNECTED]" {
		t.Errorf("wake signal text mismatch: %v", parts[0])
	}
	if cc["turnComplete"] != true {
		t.Errorf("wake signal must have turnComplete=true")
	}
}

// TestLLM_Open_NoWakeSignalWhenAbsent — when Extra has no wake_signal
// (and no history), Open must send only the setup message and nothing
// else. The model stays silent until the caller speaks.
func TestLLM_Open_NoWakeSignalWhenAbsent(t *testing.T) {
	writes := make(chan []byte, 5)
	llm, cleanup := dialMockGemini(t, func(conn *websocket.Conn) {
		_, _, _ = conn.ReadMessage() // setup (consumed, not recorded)
		_ = conn.WriteJSON(GeminiServerMessage{SetupComplete: &struct{}{}})
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			writes <- raw
		}
	})
	defer cleanup()

	err := llm.Open(context.Background(), pipeline.SetupRequest{
		SystemPrompt: "you are a test agent",
		Extra:        map[string]any{}, // no wake_signal
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	close(writes)
	var got [][]byte
	for raw := range writes {
		got = append(got, raw)
	}
	if len(got) != 0 {
		t.Fatalf("expected no client writes after setup, got %d: %s", len(got), got)
	}
}

// TestLLM_Open_GreetingTriggerNoLongerSent — regression guard for
// saffar's 2026-05-13 role-inversion bug. The old greeting_trigger key
// must not produce any realtimeInput (or any other write) after setup.
func TestLLM_Open_GreetingTriggerNoLongerSent(t *testing.T) {
	writes := make(chan []byte, 5)
	llm, cleanup := dialMockGemini(t, func(conn *websocket.Conn) {
		_, _, _ = conn.ReadMessage() // setup (consumed, not recorded)
		_ = conn.WriteJSON(GeminiServerMessage{SetupComplete: &struct{}{}})
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			writes <- raw
		}
	})
	defer cleanup()

	err := llm.Open(context.Background(), pipeline.SetupRequest{
		SystemPrompt: "you are a test agent",
		Extra: map[string]any{
			"greeting_trigger": "أهلاً وسهلاً",
		},
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	close(writes)
	var got [][]byte
	for raw := range writes {
		got = append(got, raw)
	}
	for i, w := range got {
		var m map[string]any
		_ = json.Unmarshal(w, &m)
		if _, isRealtime := m["realtimeInput"]; isRealtime {
			t.Errorf("write %d is realtimeInput (greeting_trigger reactivated): %s", i, w)
		}
	}
	if len(got) != 0 {
		t.Errorf("expected no client writes after setup (greeting_trigger ignored), got %d: %s", len(got), got)
	}
}
