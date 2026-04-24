package fake

import (
	"context"
	"sync"

	"github.com/mahdi-awadi/gopkg/voice/pipeline"
)

// LLM is a scriptable pipeline.LLM for tests.
type LLM struct {
	inFormat  pipeline.AudioFormat
	outFormat pipeline.AudioFormat

	mu sync.Mutex

	// scripted events
	eventScript  []pipeline.LLMEvent
	eventsClosed bool

	// recordings
	audioIn        []pipeline.Frame
	toolResultsIn  [][]pipeline.ToolResult
	injectedTurns  []pipeline.HistoryTurn
	opened         int
	setupCalled    pipeline.SetupRequest
	openErr        error
	sendAudioErr   error
	sendResultsErr error
}

// NewLLM builds an LLM fake with the given formats.
func NewLLM(inbound, outbound pipeline.AudioFormat) *LLM {
	return &LLM{inFormat: inbound, outFormat: outbound}
}

// Script queues events to be delivered through Events.
func (l *LLM) Script(events ...pipeline.LLMEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.eventScript = append(l.eventScript, events...)
}

// CloseEvents signals "LLM session ended" — Events' channels close
// cleanly once the scripted events drain.
func (l *LLM) CloseEvents() {
	l.mu.Lock()
	l.eventsClosed = true
	l.mu.Unlock()
}

// SetOpenErr makes the next Open return err.
func (l *LLM) SetOpenErr(err error) { l.mu.Lock(); l.openErr = err; l.mu.Unlock() }

// SetSendAudioErr makes the next SendAudio return err.
func (l *LLM) SetSendAudioErr(err error) { l.mu.Lock(); l.sendAudioErr = err; l.mu.Unlock() }

// SetSendResultsErr makes the next SendToolResults return err.
func (l *LLM) SetSendResultsErr(err error) { l.mu.Lock(); l.sendResultsErr = err; l.mu.Unlock() }

// AudioIn returns a snapshot of frames passed to SendAudio.
func (l *LLM) AudioIn() []pipeline.Frame {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]pipeline.Frame, len(l.audioIn))
	copy(out, l.audioIn)
	return out
}

// ToolResultsIn returns a snapshot of tool-result batches passed
// to SendToolResults. Outer slice: one entry per call.
func (l *LLM) ToolResultsIn() [][]pipeline.ToolResult {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([][]pipeline.ToolResult, len(l.toolResultsIn))
	for i, b := range l.toolResultsIn {
		cp := make([]pipeline.ToolResult, len(b))
		copy(cp, b)
		out[i] = cp
	}
	return out
}

// InjectedTurns returns a snapshot of turns passed to InjectTurn.
func (l *LLM) InjectedTurns() []pipeline.HistoryTurn {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]pipeline.HistoryTurn, len(l.injectedTurns))
	copy(out, l.injectedTurns)
	return out
}

// LastSetup returns the SetupRequest from the most recent Open call.
func (l *LLM) LastSetup() pipeline.SetupRequest {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.setupCalled
}

// InboundFormat implements pipeline.LLM.
func (l *LLM) InboundFormat() pipeline.AudioFormat { return l.inFormat }

// OutboundFormat implements pipeline.LLM.
func (l *LLM) OutboundFormat() pipeline.AudioFormat { return l.outFormat }

// Open records the setup and fires OnHistoryInjected events through
// the recorded path (pipeline handles that itself — fake just stores).
func (l *LLM) Open(_ context.Context, setup pipeline.SetupRequest) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.opened++
	l.setupCalled = setup
	if l.openErr != nil {
		err := l.openErr
		l.openErr = nil
		return err
	}
	// History flush is simulated by appending to injectedTurns.
	l.injectedTurns = append(l.injectedTurns, setup.History...)
	return nil
}

// SendAudio records the frame.
func (l *LLM) SendAudio(_ context.Context, f pipeline.Frame) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.sendAudioErr != nil {
		err := l.sendAudioErr
		l.sendAudioErr = nil
		return err
	}
	l.audioIn = append(l.audioIn, f)
	return nil
}

// SendToolResults records the batch.
func (l *LLM) SendToolResults(_ context.Context, results []pipeline.ToolResult) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.sendResultsErr != nil {
		err := l.sendResultsErr
		l.sendResultsErr = nil
		return err
	}
	cp := make([]pipeline.ToolResult, len(results))
	copy(cp, results)
	l.toolResultsIn = append(l.toolResultsIn, cp)
	return nil
}

// InjectTurn records the turn.
func (l *LLM) InjectTurn(_ context.Context, turn pipeline.HistoryTurn) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.injectedTurns = append(l.injectedTurns, turn)
	return nil
}

// Events delivers scripted events then closes.
func (l *LLM) Events(ctx context.Context) (<-chan pipeline.LLMEvent, <-chan error) {
	events := make(chan pipeline.LLMEvent)
	errs := make(chan error, 1)
	go func() {
		defer close(events)
		defer close(errs)
		for {
			l.mu.Lock()
			if len(l.eventScript) > 0 {
				ev := l.eventScript[0]
				l.eventScript = l.eventScript[1:]
				l.mu.Unlock()
				select {
				case events <- ev:
				case <-ctx.Done():
					return
				}
				continue
			}
			closed := l.eventsClosed
			l.mu.Unlock()
			if closed {
				return
			}
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()
	return events, errs
}

// Close is idempotent and records nothing for v0.1.0.
func (l *LLM) Close() error { return nil }

var _ pipeline.LLM = (*LLM)(nil)
