package toolexec

import (
	"context"
	"errors"
	"testing"

	"github.com/mahdi-awadi/gopkg/voice/pipeline"
)

func TestExecute_DispatchesToFunc(t *testing.T) {
	called := false
	var gotCall pipeline.ToolCall
	var gotSess pipeline.Session

	exec := New(func(ctx context.Context, call pipeline.ToolCall, sess pipeline.Session) (any, error) {
		called = true
		gotCall = call
		gotSess = sess
		return map[string]any{"ok": true}, nil
	})

	res, err := exec.Execute(
		context.Background(),
		pipeline.ToolCall{ID: "c1", Name: "search_catalog", Args: map[string]any{"q": "wrench"}},
		pipeline.Session{ID: "s1", Attrs: map[string]string{"k": "v"}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("dispatch func was not invoked")
	}
	if gotCall.Name != "search_catalog" {
		t.Errorf("unexpected call.Name = %q", gotCall.Name)
	}
	if gotSess.ID != "s1" {
		t.Errorf("unexpected sess.ID = %q", gotSess.ID)
	}
	m, ok := res.(map[string]any)
	if !ok || m["ok"] != true {
		t.Errorf("unexpected result: %#v", res)
	}
}

func TestExecute_PropagatesError(t *testing.T) {
	want := errors.New("boom")
	exec := New(func(_ context.Context, _ pipeline.ToolCall, _ pipeline.Session) (any, error) {
		return nil, want
	})
	_, got := exec.Execute(context.Background(), pipeline.ToolCall{Name: "x"}, pipeline.Session{})
	if !errors.Is(got, want) {
		t.Errorf("expected wrapped err, got %v", got)
	}
}

func TestExecute_NilFuncReturnsError(t *testing.T) {
	exec := New(nil)
	_, err := exec.Execute(context.Background(), pipeline.ToolCall{Name: "x"}, pipeline.Session{})
	if err == nil {
		t.Fatal("expected error for nil dispatch func")
	}
}

func TestExecutor_ImplementsPipelineToolExecutor(t *testing.T) {
	var _ pipeline.ToolExecutor = (*Executor)(nil)
}

// TestExecute_MissingAttrsTolerated preserves the original test's intent:
// confirm that empty pipeline.Session.Attrs flows through cleanly. In the
// new Func shape, the dispatch function (not the executor) decides how to
// read attrs; we just verify the executor does not synthesize defaults or
// error on an empty attr map.
func TestExecute_MissingAttrsTolerated(t *testing.T) {
	var sawAttrs map[string]string
	exec := New(func(_ context.Context, _ pipeline.ToolCall, sess pipeline.Session) (any, error) {
		sawAttrs = sess.Attrs
		return "ok", nil
	})
	_, err := exec.Execute(context.Background(), pipeline.ToolCall{Name: "x"}, pipeline.Session{})
	if err != nil {
		t.Errorf("Execute err on empty attrs: %v", err)
	}
	if len(sawAttrs) != 0 {
		t.Errorf("expected empty attrs, got %v", sawAttrs)
	}
}
