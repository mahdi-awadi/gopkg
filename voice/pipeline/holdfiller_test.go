package pipeline

import (
	"context"
	"testing"
)

func TestSilentFiller_ReturnsClosedChannel(t *testing.T) {
	f := SilentFiller{}
	ch := f.Frames(context.Background())
	if ch == nil {
		t.Fatal("SilentFiller.Frames returned nil")
	}
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("SilentFiller channel must be closed (produces zero frames)")
		}
	default:
		t.Error("SilentFiller channel should be immediately readable as closed")
	}
}

func TestSilentFiller_ImplementsHoldFiller(t *testing.T) {
	var _ HoldFiller = SilentFiller{}
}
