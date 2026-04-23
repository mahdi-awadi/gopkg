// Package pipeline provides a generic filter-chain executor.
//
// A Pipeline[T] runs a sequence of Filter[T]s over a slice of items.
// Each filter can transform, drop, or enrich items. Filters are
// classified by Criticality (whether a failure aborts the pipeline
// or is swallowed) and by Phase (a tag the caller can use to run a
// subset of filters at a time).
//
// The package has zero third-party dependencies. Observability is
// surfaced through a small Logger interface (implementations for
// zap/slog/etc. are trivial) and through a slice of StepLog records
// returned from every Run, so callers can attach them to traces or
// return them in API responses.
package pipeline

import (
	"context"
	"fmt"
	"time"
)

// Criticality distinguishes filters that must succeed from those that
// are safe to skip on error.
type Criticality int

const (
	// Critical filters abort the pipeline when they return an error.
	Critical Criticality = iota
	// Enrichment filters are skipped when they error; the pipeline
	// continues with the pre-filter item set.
	Enrichment
)

// Phase tags a filter with the logical stage it belongs in. Pipeline
// itself does not interpret the value; callers use RunPhase to run a
// subset at a time (e.g. per-batch vs. post-accumulation).
type Phase int

// Filter is a single processing stage in the pipeline.
//
// Apply receives the current slice of items and an opaque metadata
// value the caller threaded through Run / RunPhase; it returns a
// (possibly reduced, possibly enriched) slice.
type Filter[T any] interface {
	Name() string
	Criticality() Criticality
	Phase() Phase
	Apply(ctx context.Context, items []T, meta any) ([]T, error)
}

// StepLog records what a single filter did.
// Populated by Pipeline on every run, regardless of outcome.
type StepLog struct {
	FilterName  string
	Criticality Criticality
	Phase       Phase
	BeforeCount int
	AfterCount  int
	DurationMs  int64
	Removed     int
	Skipped     bool   // true when an Enrichment filter errored and was skipped
	Error       string // empty on success
}

// Logger is the minimum observability contract. Implementations can
// wrap *zap.Logger, *slog.Logger, or any structured logger.
//
// The zero value (NoopLogger) is a valid implementation.
type Logger interface {
	Debug(msg string, fields map[string]any)
	Warn(msg string, fields map[string]any)
	Error(msg string, fields map[string]any)
}

// NoopLogger discards every call.
type NoopLogger struct{}

// Debug discards the call.
func (NoopLogger) Debug(string, map[string]any) {}

// Warn discards the call.
func (NoopLogger) Warn(string, map[string]any) {}

// Error discards the call.
func (NoopLogger) Error(string, map[string]any) {}

// Pipeline executes filters in registration order.
// Safe for concurrent Run / RunPhase calls (Pipeline itself is
// immutable after New; filters must be safe for concurrent Apply
// if the pipeline is shared).
type Pipeline[T any] struct {
	filters []Filter[T]
	logger  Logger
}

// New creates a Pipeline from a variadic list of filters.
// If logger is nil, NoopLogger{} is used.
func New[T any](logger Logger, filters ...Filter[T]) *Pipeline[T] {
	if logger == nil {
		logger = NoopLogger{}
	}
	// Copy to defend against caller mutations after construction.
	cp := make([]Filter[T], len(filters))
	copy(cp, filters)
	return &Pipeline[T]{filters: cp, logger: logger}
}

// Run executes every registered filter in order.
// Critical failure returns a non-nil error along with the logs
// collected so far. Enrichment failures are swallowed; the pipeline
// continues with the pre-filter slice.
func (p *Pipeline[T]) Run(ctx context.Context, items []T, meta any) ([]T, []StepLog, error) {
	logs := make([]StepLog, 0, len(p.filters))
	current := items

	for _, f := range p.filters {
		start := time.Now()
		before := len(current)
		result, err := f.Apply(ctx, current, meta)
		elapsed := time.Since(start)

		if err != nil {
			entry := StepLog{
				FilterName:  f.Name(),
				Criticality: f.Criticality(),
				Phase:       f.Phase(),
				BeforeCount: before,
				AfterCount:  before,
				DurationMs:  elapsed.Milliseconds(),
				Error:       err.Error(),
			}

			if f.Criticality() == Critical {
				logs = append(logs, entry)
				p.logger.Error("pipeline: critical filter failed, aborting", map[string]any{
					"filter": f.Name(),
					"error":  err.Error(),
				})
				return nil, logs, fmt.Errorf("critical filter %s failed: %w", f.Name(), err)
			}

			entry.Skipped = true
			logs = append(logs, entry)
			p.logger.Warn("pipeline: enrichment filter failed, skipping", map[string]any{
				"filter": f.Name(),
				"error":  err.Error(),
			})
			continue
		}

		logs = append(logs, StepLog{
			FilterName:  f.Name(),
			Criticality: f.Criticality(),
			Phase:       f.Phase(),
			BeforeCount: before,
			AfterCount:  len(result),
			DurationMs:  elapsed.Milliseconds(),
			Removed:     before - len(result),
		})
		p.logger.Debug("pipeline: step complete", map[string]any{
			"filter": f.Name(),
			"before": before,
			"after":  len(result),
			"ms":     elapsed.Milliseconds(),
		})
		current = result
	}
	return current, logs, nil
}

// RunPhase executes only the filters whose Phase() matches `phase`,
// in registration order. Equivalent to Run on a subset.
func (p *Pipeline[T]) RunPhase(ctx context.Context, phase Phase, items []T, meta any) ([]T, []StepLog, error) {
	subset := make([]Filter[T], 0, len(p.filters))
	for _, f := range p.filters {
		if f.Phase() == phase {
			subset = append(subset, f)
		}
	}
	sub := &Pipeline[T]{filters: subset, logger: p.logger}
	return sub.Run(ctx, items, meta)
}

// Filters returns a copy of the registered filter list.
// Useful for inspection and for building adjacent pipelines.
func (p *Pipeline[T]) Filters() []Filter[T] {
	out := make([]Filter[T], len(p.filters))
	copy(out, p.filters)
	return out
}
