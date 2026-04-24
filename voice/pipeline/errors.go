package pipeline

import "errors"

// ErrFormatBridge is returned by Run when the Transport/LLM format
// pair can't be resolved by the internal codec bridge.
var ErrFormatBridge = errors.New("pipeline: no codec bridge for format pair")

// ErrToolExecutorPanicked wraps a panic recovered inside
// ToolExecutor.Execute. Surfaced as ToolResult.Err; session continues.
var ErrToolExecutorPanicked = errors.New("pipeline: tool executor panicked")
