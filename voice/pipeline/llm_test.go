package pipeline

import "testing"

func TestLLMEvents_AllImplementInterface(t *testing.T) {
	events := []LLMEvent{
		EventAudioOut{},
		EventAssistantText{},
		EventCallerTranscript{},
		EventTurnComplete{},
		EventInterrupted{},
		EventToolCalls{},
	}
	if len(events) == 0 {
		t.Fatal("no events declared")
	}
	// Compile-time check is via the slice — if any didn't
	// implement LLMEvent this test file wouldn't compile.
	for i, e := range events {
		if e == nil {
			t.Errorf("events[%d] is nil", i)
		}
	}
}
