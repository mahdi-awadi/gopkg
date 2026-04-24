package pipeline

import "testing"

func TestEncodingConstants(t *testing.T) {
	if EncodingMulaw != "mulaw" {
		t.Errorf("EncodingMulaw=%q, want %q", EncodingMulaw, "mulaw")
	}
	if EncodingPCM16LE != "pcm16le" {
		t.Errorf("EncodingPCM16LE=%q, want %q", EncodingPCM16LE, "pcm16le")
	}
}

func TestRoleConstants(t *testing.T) {
	if RoleUser != "user" || RoleAssistant != "assistant" {
		t.Errorf("Role constants wrong: %q, %q", RoleUser, RoleAssistant)
	}
}

func TestEndReasonConstants(t *testing.T) {
	want := map[EndReason]string{
		EndReasonTransportClosed: "transport_closed",
		EndReasonLLMClosed:       "llm_closed",
		EndReasonContextDone:     "context_done",
		EndReasonFatalError:      "fatal_error",
	}
	for r, s := range want {
		if string(r) != s {
			t.Errorf("EndReason(%q)=%q", s, r)
		}
	}
}

func TestAudioFormatStruct(t *testing.T) {
	f := AudioFormat{Encoding: EncodingPCM16LE, SampleRate: 16000, Channels: 1}
	if f.Encoding != EncodingPCM16LE || f.SampleRate != 16000 || f.Channels != 1 {
		t.Errorf("AudioFormat fields not assigned: %+v", f)
	}
}
