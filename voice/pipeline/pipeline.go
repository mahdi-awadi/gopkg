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

	// cancel shuts down all goroutines by canceling runCtx. It is called on:
	//   (a) outer ctx cancellation (via runCtx being a child of ctx),
	//   (b) fatal error in either goroutine,
	//   (c) after wg.Wait() when both goroutines exit naturally.
	// It is NOT called on transport close alone, so in-flight LLM events
	// can be fully dispatched before the pipeline shuts down.
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

	var wg sync.WaitGroup
	wg.Add(2)

	var (
		runErr    error
		endReason = EndReasonContextDone
		endMu     sync.Mutex
		endOnce   sync.Once
	)

	// recordEnd captures the first terminal event. It does NOT cancel runCtx;
	// fatal paths call cancel() explicitly.
	recordEnd := func(r EndReason, e error) {
		endOnce.Do(func() {
			endMu.Lock()
			if ctx.Err() != nil {
				endReason = EndReasonContextDone
				runErr = ctx.Err()
			} else {
				endReason = r
				runErr = e
			}
			endMu.Unlock()
		})
	}

	// Caller-audio goroutine: Transport.Receive → bridge → LLM.SendAudio.
	// Exits when inFrames closes (covers transport close and ctx cancellation).
	// Does NOT check runCtx.Done — it waits for inFrames to close naturally.
	// Fatal errors call cancel() to trigger shutdown of the other goroutine.
	go func() {
		defer wg.Done()
		for {
			select {
			case f, ok := <-inFrames:
				if !ok {
					recordEnd(EndReasonTransportClosed, nil)
					return
				}
				conv, err := inboundBridge(f)
				if err != nil {
					recordEnd(EndReasonFatalError, err)
					p.safeObserve(func() { p.opts.Observer.OnError(ctx, sess, err) })
					cancel()
					return
				}
				if err := llm.SendAudio(runCtx, conv); err != nil {
					recordEnd(EndReasonFatalError, err)
					p.safeObserve(func() { p.opts.Observer.OnError(ctx, sess, err) })
					cancel()
					return
				}
			case err, ok := <-inErrs:
				if ok && err != nil {
					recordEnd(EndReasonFatalError, err)
					p.safeObserve(func() { p.opts.Observer.OnError(ctx, sess, err) })
					cancel()
					return
				}
			}
		}
	}()

	// LLM-events goroutine: dispatches LLM events.
	// Exits when llmEvents closes (covers LLM close and ctx cancellation).
	// Does NOT check runCtx.Done — it waits for llmEvents to close naturally.
	// Fatal errors call cancel() to trigger shutdown of the other goroutine.
	go func() {
		defer wg.Done()
		turn := 0
		for {
			select {
			case ev, ok := <-llmEvents:
				if !ok {
					recordEnd(EndReasonLLMClosed, nil)
					return
				}
				if err := p.dispatchLLMEvent(ctx, runCtx, sess, transport, llm, executor, outboundBridge, ev, &turn); err != nil {
					recordEnd(EndReasonFatalError, err)
					p.safeObserve(func() { p.opts.Observer.OnError(ctx, sess, err) })
					cancel()
					return
				}
			case err, ok := <-llmErrs:
				if ok && err != nil {
					recordEnd(EndReasonFatalError, err)
					p.safeObserve(func() { p.opts.Observer.OnError(ctx, sess, err) })
					cancel()
					return
				}
			}
		}
	}()

	// Wait for both goroutines to exit naturally, OR for the outer context
	// to be canceled. In the latter case, cancel runCtx to signal the goroutines
	// to stop (which causes their channels to close via the LLM/transport adapters).
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Both goroutines exited naturally; nothing more to do.
	case <-ctx.Done():
		cancel() // trigger goroutine shutdown via runCtx
		<-done   // wait for goroutines to exit
	}

	endMu.Lock()
	reason := endReason
	rErr := runErr
	endMu.Unlock()

	if errors.Is(runCtx.Err(), context.Canceled) && rErr == nil && reason == EndReasonContextDone {
		rErr = runCtx.Err()
	}

	// Cleanup + OnSessionEnd.
	_ = transport.Close()
	_ = llm.Close()
	p.safeObserve(func() { p.opts.Observer.OnSessionEnd(ctx, sess, reason) })
	return rErr
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

	case EventAssistantText:
		p.safeObserve(func() { p.opts.Observer.OnAssistantText(ctx, sess, e.Text, e.Final) })
		return nil

	case EventCallerTranscript:
		p.safeObserve(func() { p.opts.Observer.OnCallerTranscript(ctx, sess, e.Text) })
		return nil

	case EventTurnComplete:
		*turn++
		name := "turn-" + itoa(*turn)
		_ = transport.Mark(runCtx, name) // non-fatal
		current := *turn
		p.safeObserve(func() { p.opts.Observer.OnTurnComplete(ctx, sess, current) })
		return nil

	case EventInterrupted:
		_ = transport.Clear(runCtx) // non-fatal
		p.safeObserve(func() { p.opts.Observer.OnInterrupted(ctx, sess) })
		return nil

	case EventToolCalls:
		p.dispatchToolCalls(ctx, runCtx, sess, transport, llm, executor, e.Calls)
		return nil
	}
	return nil
}

// dispatchToolCalls executes a batch of tool calls with concurrency
// bounded by Options.ToolConcurrency, streams the HoldFiller after
// HoldFillerDelay, and flushes results to the LLM in call-order.
func (p *Pipeline) dispatchToolCalls(
	ctx, runCtx context.Context,
	sess Session,
	transport Transport,
	llm LLM,
	executor ToolExecutor,
	calls []ToolCall,
) {
	// Fire OnToolCall for every call in order.
	for _, c := range calls {
		call := c
		p.safeObserve(func() { p.opts.Observer.OnToolCall(ctx, sess, call) })
	}

	results := make([]ToolResult, len(calls))
	done := make(chan struct{})
	holdCtx, holdCancel := context.WithCancel(runCtx)

	// Hold-filler watchdog.
	go func() {
		select {
		case <-done:
			return
		case <-time.After(p.opts.HoldFillerDelay):
			p.pumpHoldFiller(holdCtx, transport)
		}
	}()

	sem := make(chan struct{}, p.opts.ToolConcurrency)
	var wg sync.WaitGroup
	for i, c := range calls {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, call ToolCall) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					results[idx] = ToolResult{
						CallID: call.ID,
						Err:    fmt.Errorf("%w: %v", ErrToolExecutorPanicked, r),
					}
				}
			}()
			data, err := executor.Execute(runCtx, call, sess)
			results[idx] = ToolResult{CallID: call.ID, Data: data, Err: err}
		}(i, c)
	}
	wg.Wait()
	close(done)
	holdCancel()
	_ = transport.Clear(runCtx) // flush any leftover filler audio

	// OnToolResponse in call-order.
	for i := range calls {
		call := calls[i]
		res := results[i]
		p.safeObserve(func() { p.opts.Observer.OnToolResponse(ctx, sess, call, res.Data, res.Err) })
	}

	if err := llm.SendToolResults(runCtx, results); err != nil {
		p.safeObserve(func() { p.opts.Observer.OnError(ctx, sess, err) })
	}
}

// pumpHoldFiller reads HoldFiller.Frames(holdCtx) and forwards every
// frame to Transport.Send. Exits when the filler channel closes or
// holdCtx is cancelled.
func (p *Pipeline) pumpHoldFiller(holdCtx context.Context, transport Transport) {
	ch := p.opts.Filler.Frames(holdCtx)
	for {
		select {
		case <-holdCtx.Done():
			return
		case f, ok := <-ch:
			if !ok {
				return
			}
			if err := transport.Send(holdCtx, f); err != nil {
				return
			}
		}
	}
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
