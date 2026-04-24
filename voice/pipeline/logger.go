package pipeline

// Logger is the minimum structured-logging contract the pipeline uses.
// Wrappers for *zap.Logger / *slog.Logger live in consumer code.
// The zero value (NoopLogger{}) is safe.
type Logger interface {
	Debug(msg string, fields map[string]any)
	Info(msg string, fields map[string]any)
	Warn(msg string, fields map[string]any)
	Error(msg string, fields map[string]any)
}

// NoopLogger discards every call. Zero-value ready.
type NoopLogger struct{}

// Debug discards the call.
func (NoopLogger) Debug(string, map[string]any) {}

// Info discards the call.
func (NoopLogger) Info(string, map[string]any) {}

// Warn discards the call.
func (NoopLogger) Warn(string, map[string]any) {}

// Error discards the call.
func (NoopLogger) Error(string, map[string]any) {}
