package pipeline

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mahdi-awadi/gopkg/id"
)

// Options configures a Pipeline. The zero value is not valid — use
// New, which applies sane defaults for every zero field.
type Options struct {
	ToolConcurrency int
	HoldFillerDelay time.Duration
	Filler          HoldFiller
	Observer        Observer
	Logger          Logger
	SessionIDFunc   func() string
}

// Pipeline is the orchestrator. Safe for concurrent Run calls with
// different Transport / LLM / Executor tuples. NOT safe to Run twice
// with the same Transport or LLM (both get Close()d).
type Pipeline struct {
	opts Options
}

// New constructs a Pipeline. Rejects negative ToolConcurrency or
// HoldFillerDelay; applies defaults for zero-valued fields.
func New(opts Options) (*Pipeline, error) {
	if opts.ToolConcurrency < 0 {
		return nil, fmt.Errorf("pipeline.New: ToolConcurrency=%d must be >= 0", opts.ToolConcurrency)
	}
	if opts.HoldFillerDelay < 0 {
		return nil, fmt.Errorf("pipeline.New: HoldFillerDelay=%v must be >= 0", opts.HoldFillerDelay)
	}
	if opts.ToolConcurrency == 0 {
		opts.ToolConcurrency = 1
	}
	if opts.HoldFillerDelay == 0 {
		opts.HoldFillerDelay = 2 * time.Second
	}
	if opts.Filler == nil {
		opts.Filler = SilentFiller{}
	}
	if opts.Observer == nil {
		opts.Observer = NoopObserver{}
	}
	if opts.Logger == nil {
		opts.Logger = NoopLogger{}
	}
	if opts.SessionIDFunc == nil {
		opts.SessionIDFunc = id.UUIDv7
	}
	return &Pipeline{opts: opts}, nil
}

// Run orchestrates one complete voice session. See the package doc
// and the design spec for full semantics.
func (p *Pipeline) Run(
	ctx context.Context,
	transport Transport,
	llm LLM,
	executor ToolExecutor,
	setup SetupRequest,
	attrs map[string]string,
) error {
	// Build the Session struct.
	sess := Session{
		ID:        p.opts.SessionIDFunc(),
		StartedAt: time.Now(),
		Attrs:     copyAttrs(attrs),
	}

	// Resolve codec bridges BEFORE any side effect. Unknown pair →
	// fire OnError + OnSessionEnd(FatalError); Run returns wrapped err.
	inboundBridge, err := resolveBridge(transport.InboundFormat(), llm.OutboundFormat())
	if err != nil {
		p.safeObserve(func() { p.opts.Observer.OnError(ctx, sess, err) })
		p.safeObserve(func() { p.opts.Observer.OnSessionEnd(ctx, sess, EndReasonFatalError) })
		return err
	}
	outboundBridge, err := resolveBridge(llm.InboundFormat(), transport.OutboundFormat())
	if err != nil {
		p.safeObserve(func() { p.opts.Observer.OnError(ctx, sess, err) })
		p.safeObserve(func() { p.opts.Observer.OnSessionEnd(ctx, sess, EndReasonFatalError) })
		return err
	}

	// Derive an internal context so we can cancel on early termination.
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Open LLM (fires history injection internally).
	if err := llm.Open(runCtx, setup); err != nil {
		p.safeObserve(func() { p.opts.Observer.OnError(ctx, sess, err) })
		p.safeObserve(func() { p.opts.Observer.OnSessionEnd(ctx, sess, EndReasonFatalError) })
		return err
	}
	p.safeObserve(func() { p.opts.Observer.OnHistoryInjected(ctx, sess, len(setup.History)) })

	// Start transport + LLM event streams.
	inFrames, inErrs := transport.Receive(runCtx)
	llmEvents, llmErrs := llm.Events(runCtx)

	// SessionStart fires once everything is ready to flow.
	p.safeObserve(func() { p.opts.Observer.OnSessionStart(ctx, sess) })

	// Spawn the caller-audio goroutine: Transport.Receive → bridge → LLM.SendAudio.
	var wg sync.WaitGroup
	wg.Add(2)

	runErr := error(nil)
	endReason := EndReasonContextDone
	var endOnce sync.Once
	setEnd := func(r EndReason, e error) {
		endOnce.Do(func() {
			endReason = r
			runErr = e
			cancel()
		})
	}

	go func() {
		defer wg.Done()
		for {
			select {
			case <-runCtx.Done():
				return
			case f, ok := <-inFrames:
				if !ok {
					setEnd(EndReasonTransportClosed, nil)
					return
				}
				conv, err := inboundBridge(f)
				if err != nil {
					setEnd(EndReasonFatalError, err)
					p.safeObserve(func() { p.opts.Observer.OnError(ctx, sess, err) })
					return
				}
				if err := llm.SendAudio(runCtx, conv); err != nil {
					setEnd(EndReasonFatalError, err)
					p.safeObserve(func() { p.opts.Observer.OnError(ctx, sess, err) })
					return
				}
			case err, ok := <-inErrs:
				if ok && err != nil {
					setEnd(EndReasonFatalError, err)
					p.safeObserve(func() { p.opts.Observer.OnError(ctx, sess, err) })
					return
				}
			}
		}
	}()

	// Spawn the LLM-events goroutine.
	go func() {
		defer wg.Done()
		turn := 0
		for {
			select {
			case <-runCtx.Done():
				return
			case ev, ok := <-llmEvents:
				if !ok {
					setEnd(EndReasonLLMClosed, nil)
					return
				}
				if err := p.dispatchLLMEvent(ctx, runCtx, sess, transport, llm, executor, outboundBridge, ev, &turn); err != nil {
					setEnd(EndReasonFatalError, err)
					p.safeObserve(func() { p.opts.Observer.OnError(ctx, sess, err) })
					return
				}
			case err, ok := <-llmErrs:
				if ok && err != nil {
					setEnd(EndReasonFatalError, err)
					p.safeObserve(func() { p.opts.Observer.OnError(ctx, sess, err) })
					return
				}
			}
		}
	}()

	<-runCtx.Done()
	wg.Wait()

	if errors.Is(runCtx.Err(), context.Canceled) && runErr == nil && endReason == EndReasonContextDone {
		runErr = runCtx.Err()
	}

	// Cleanup + OnSessionEnd.
	_ = transport.Close()
	_ = llm.Close()
	p.safeObserve(func() { p.opts.Observer.OnSessionEnd(ctx, sess, endReason) })
	return runErr
}

// safeObserve wraps an observer callback in panic recovery.
func (p *Pipeline) safeObserve(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			p.opts.Logger.Error("pipeline: observer panic", map[string]any{"panic": r})
		}
	}()
	fn()
}

// dispatchLLMEvent handles one LLMEvent emitted by the LLM. Returns a
// non-nil error only for fatal conditions (Transport.Send failure).
func (p *Pipeline) dispatchLLMEvent(
	ctx, runCtx context.Context,
	sess Session,
	transport Transport,
	llm LLM,
	executor ToolExecutor,
	outboundBridge bridgeFn,
	ev LLMEvent,
	turn *int,
) error {
	switch e := ev.(type) {
	case EventAudioOut:
		conv, err := outboundBridge(e.Frame)
		if err != nil {
			return err
		}
		return transport.Send(runCtx, conv)
	}
	return nil
}

func copyAttrs(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	cp := make(map[string]string, len(src))
	for k, v := range src {
		cp[k] = v
	}
	return cp
}
