package pipeline

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// testFilter is a minimal Filter[int] for testing.
type testFilter struct {
	name        string
	criticality Criticality
	phase       Phase
	apply       func(ctx context.Context, items []int, meta any) ([]int, error)
}

func (f *testFilter) Name() string               { return f.name }
func (f *testFilter) Criticality() Criticality   { return f.criticality }
func (f *testFilter) Phase() Phase               { return f.phase }
func (f *testFilter) Apply(ctx context.Context, items []int, meta any) ([]int, error) {
	return f.apply(ctx, items, meta)
}

func TestPipeline_RunEmpty(t *testing.T) {
	p := New[int](nil)
	out, logs, err := p.Run(context.Background(), []int{1, 2, 3}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 3 || len(logs) != 0 {
		t.Fatalf("empty pipeline should pass through; got %v / logs %v", out, logs)
	}
}

func TestPipeline_RunFiltersInOrder(t *testing.T) {
	removeEven := &testFilter{
		name: "remove-even",
		apply: func(_ context.Context, xs []int, _ any) ([]int, error) {
			out := xs[:0]
			for _, x := range xs {
				if x%2 == 1 {
					out = append(out, x)
				}
			}
			return out, nil
		},
	}
	double := &testFilter{
		name: "double",
		apply: func(_ context.Context, xs []int, _ any) ([]int, error) {
			out := make([]int, len(xs))
			for i, x := range xs {
				out[i] = x * 2
			}
			return out, nil
		},
	}
	p := New[int](nil, removeEven, double)
	out, logs, err := p.Run(context.Background(), []int{1, 2, 3, 4, 5}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 3 || out[0] != 2 || out[1] != 6 || out[2] != 10 {
		t.Errorf("got %v, want [2 6 10]", out)
	}
	if len(logs) != 2 {
		t.Fatalf("want 2 logs, got %d", len(logs))
	}
	if logs[0].Removed != 2 || logs[1].Removed != 0 {
		t.Errorf("removed counts: %d, %d; want 2, 0", logs[0].Removed, logs[1].Removed)
	}
}

func TestPipeline_CriticalFailureAborts(t *testing.T) {
	good := &testFilter{
		name: "good",
		apply: func(_ context.Context, xs []int, _ any) ([]int, error) {
			return xs, nil
		},
	}
	bad := &testFilter{
		name:        "bad",
		criticality: Critical,
		apply: func(_ context.Context, _ []int, _ any) ([]int, error) {
			return nil, errors.New("nope")
		},
	}
	downstream := &testFilter{
		name: "downstream",
		apply: func(_ context.Context, xs []int, _ any) ([]int, error) {
			return append(xs, 999), nil
		},
	}
	p := New[int](nil, good, bad, downstream)
	out, logs, err := p.Run(context.Background(), []int{1, 2}, nil)
	if err == nil {
		t.Fatalf("expected error on critical failure")
	}
	if !strings.Contains(err.Error(), "bad") {
		t.Errorf("error should mention filter name, got: %v", err)
	}
	if out != nil {
		t.Errorf("expected nil items on abort, got %v", out)
	}
	// Should contain good + bad logs, but NOT downstream.
	if len(logs) != 2 {
		t.Fatalf("want 2 logs (good + bad), got %d", len(logs))
	}
	if logs[1].Error == "" {
		t.Errorf("critical failure log should carry error string")
	}
	if logs[1].Skipped {
		t.Errorf("critical failures must not be marked Skipped")
	}
}

func TestPipeline_EnrichmentFailureSkipped(t *testing.T) {
	enrich := &testFilter{
		name:        "enrich",
		criticality: Enrichment,
		apply: func(_ context.Context, _ []int, _ any) ([]int, error) {
			return nil, errors.New("external API down")
		},
	}
	passthrough := &testFilter{
		name: "passthrough",
		apply: func(_ context.Context, xs []int, _ any) ([]int, error) {
			return xs, nil
		},
	}
	p := New[int](nil, enrich, passthrough)
	out, logs, err := p.Run(context.Background(), []int{1, 2, 3}, nil)
	if err != nil {
		t.Fatalf("enrichment failure must not surface error, got: %v", err)
	}
	if len(out) != 3 {
		t.Errorf("items should pass through untouched, got %v", out)
	}
	if len(logs) != 2 {
		t.Fatalf("want 2 logs, got %d", len(logs))
	}
	if !logs[0].Skipped {
		t.Errorf("enrichment failure should be marked Skipped")
	}
	if logs[0].AfterCount != logs[0].BeforeCount {
		t.Errorf("skipped filter should not decrement count")
	}
}

func TestPipeline_RunPhase(t *testing.T) {
	const (
		perBatch Phase = iota
		accumulation
	)
	perBatchFilter := &testFilter{
		name:  "per-batch",
		phase: perBatch,
		apply: func(_ context.Context, xs []int, _ any) ([]int, error) {
			return append(xs, -1), nil
		},
	}
	accFilter := &testFilter{
		name:  "acc",
		phase: accumulation,
		apply: func(_ context.Context, xs []int, _ any) ([]int, error) {
			return append(xs, -2), nil
		},
	}
	p := New[int](nil, perBatchFilter, accFilter)

	out, logs, err := p.RunPhase(context.Background(), perBatch, []int{0}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 2 || out[1] != -1 {
		t.Errorf("got %v, want only per-batch filter to run", out)
	}
	if len(logs) != 1 || logs[0].FilterName != "per-batch" {
		t.Errorf("expected 1 log for per-batch only, got %v", logs)
	}
}

func TestPipeline_MetaPassthrough(t *testing.T) {
	type metaShape struct{ Tag string }
	seen := ""
	capture := &testFilter{
		name: "capture",
		apply: func(_ context.Context, xs []int, meta any) ([]int, error) {
			if m, ok := meta.(metaShape); ok {
				seen = m.Tag
			}
			return xs, nil
		},
	}
	p := New[int](nil, capture)
	_, _, err := p.Run(context.Background(), nil, metaShape{Tag: "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seen != "hello" {
		t.Errorf("meta not passed through; got %q", seen)
	}
}

func TestPipeline_FiltersReturnsCopy(t *testing.T) {
	f := &testFilter{name: "x"}
	p := New[int](nil, f)
	out := p.Filters()
	out[0] = &testFilter{name: "mutated"}
	again := p.Filters()
	if again[0].Name() != "x" {
		t.Errorf("Filters() should return a copy, got mutated list")
	}
}

func TestPipeline_ConstructorCopiesFilters(t *testing.T) {
	orig := []Filter[int]{&testFilter{name: "a"}, &testFilter{name: "b"}}
	p := New[int](nil, orig...)
	orig[0] = &testFilter{name: "hacked"}
	if p.Filters()[0].Name() != "a" {
		t.Errorf("pipeline must defensive-copy its filter slice")
	}
}

func TestNoopLogger(t *testing.T) {
	var l Logger = NoopLogger{}
	l.Debug("", nil)
	l.Warn("", nil)
	l.Error("", nil)
}
