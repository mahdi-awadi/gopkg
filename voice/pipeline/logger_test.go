package pipeline

import "testing"

func TestNoopLogger_AllMethodsSafe(t *testing.T) {
	var l Logger = NoopLogger{}
	l.Debug("msg", nil)
	l.Info("msg", nil)
	l.Warn("msg", nil)
	l.Error("msg", map[string]any{"k": "v"})
	// pass if we got here without panic
}
