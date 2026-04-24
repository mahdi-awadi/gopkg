package pipeline

import (
	"fmt"
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
