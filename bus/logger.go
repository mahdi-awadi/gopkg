package bus

import "go.uber.org/zap"

// Logger is the minimum logging contract broker and adapters require.
//
// Consumers inject any compatible implementation. The two-method surface
// keeps the broker package free of any specific logging-framework
// dependency; a concrete adapter for *zap.Logger is provided as
// WrapZap for convenience, but consumers may pass any value that
// satisfies the interface.
type Logger interface {
	Info(msg string, fields map[string]any)
	Error(msg string, fields map[string]any)
}

// NoopLogger is a Logger that discards every call. Safe for use in
// tests or production when logging is intentionally disabled.
//
// The zero value is ready to use.
type NoopLogger struct{}

// Info discards the call.
func (NoopLogger) Info(string, map[string]any) {}

// Error discards the call.
func (NoopLogger) Error(string, map[string]any) {}

// zapAdapter adapts *zap.Logger to the Logger interface.
type zapAdapter struct {
	z *zap.Logger
}

// WrapZap returns a Logger backed by the given *zap.Logger. If z is nil,
// a no-op logger is returned.
func WrapZap(z *zap.Logger) Logger {
	if z == nil {
		return NoopLogger{}
	}
	return &zapAdapter{z: z}
}

func (a *zapAdapter) Info(msg string, fields map[string]any) {
	a.z.Info(msg, toZapFields(fields)...)
}

func (a *zapAdapter) Error(msg string, fields map[string]any) {
	a.z.Error(msg, toZapFields(fields)...)
}

func toZapFields(fields map[string]any) []zap.Field {
	if len(fields) == 0 {
		return nil
	}
	out := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		out = append(out, zap.Any(k, v))
	}
	return out
}
