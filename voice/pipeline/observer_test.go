package pipeline

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"testing"
	"time"
)

func TestNoopObserver_AllMethodsSafe(t *testing.T) {
	var o Observer = NoopObserver{}
	ctx := context.Background()
	s := Session{ID: "x", StartedAt: time.Now()}
	o.OnSessionStart(ctx, s)
	o.OnHistoryInjected(ctx, s, 0)
	o.OnCallerTranscript(ctx, s, "")
	o.OnAssistantText(ctx, s, "", true)
	o.OnToolCall(ctx, s, ToolCall{})
	o.OnToolResponse(ctx, s, ToolCall{}, nil, nil)
	o.OnInterrupted(ctx, s)
	o.OnTurnComplete(ctx, s, 1)
	o.OnError(ctx, s, errors.New("x"))
	o.OnSessionEnd(ctx, s, EndReasonContextDone)
}

// recorder captures which callback fired.
type recorder struct {
	NoopObserver
	events []string
}

func (r *recorder) OnSessionStart(context.Context, Session) { r.events = append(r.events, "start") }
func (r *recorder) OnSessionEnd(context.Context, Session, EndReason) {
	r.events = append(r.events, "end")
}
func (r *recorder) OnAssistantText(_ context.Context, _ Session, t string, _ bool) {
	r.events = append(r.events, "text:"+t)
}

func TestMulti_ForwardsToAll(t *testing.T) {
	a := &recorder{}
	b := &recorder{}
	m := Multi(a, b)
	ctx := context.Background()
	s := Session{}
	m.OnSessionStart(ctx, s)
	m.OnAssistantText(ctx, s, "hi", true)
	m.OnSessionEnd(ctx, s, EndReasonContextDone)
	want := []string{"start", "text:hi", "end"}
	sort.Strings(a.events)
	sort.Strings(b.events)
	sort.Strings(want)
	if !reflect.DeepEqual(a.events, b.events) || !reflect.DeepEqual(a.events, want) {
		t.Errorf("a=%v b=%v want=%v", a.events, b.events, want)
	}
}

func TestMulti_NilArgPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on nil observer")
		}
	}()
	Multi(NoopObserver{}, nil)
}

func TestMulti_OrderedForwarding(t *testing.T) {
	order := []string{}
	a := &recFn{fn: func(tag string) { order = append(order, "a:"+tag) }}
	b := &recFn{fn: func(tag string) { order = append(order, "b:"+tag) }}
	m := Multi(a, b)
	m.OnSessionStart(context.Background(), Session{})
	want := []string{"a:start", "b:start"}
	if !reflect.DeepEqual(order, want) {
		t.Errorf("order=%v want=%v", order, want)
	}
}

type recFn struct {
	NoopObserver
	fn func(tag string)
}

func (r *recFn) OnSessionStart(context.Context, Session) { r.fn("start") }
