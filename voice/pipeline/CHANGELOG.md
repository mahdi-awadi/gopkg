# Changelog

## [0.1.0] - 2026-04-24

### Added
- `Pipeline` orchestrator with `Options`, `New`, `Run`
- Interfaces: `Transport`, `LLM`, `ToolExecutor`, `HoldFiller`, `Observer`, `Logger`
- Core types: `AudioFormat`, `Frame`, `Session`, `HistoryTurn`, `ToolCall`, `ToolResult`, `ToolDecl`, `ToolSchema`, `ToolProperty`, `EndReason`, `SetupRequest`
- LLM event variants: `EventAudioOut`, `EventAssistantText`, `EventCallerTranscript`, `EventTurnComplete`, `EventInterrupted`, `EventToolCalls`
- `NoopObserver` (embed) + `Multi(...)` helper
- `SilentFiller` default hold-filler; `NoopLogger` default
- Codec auto-bridge via `gopkg/audio/codec` for mulaw@8k ↔ pcm16le@16k and pcm16le@24k → mulaw@8k pairs
- Tool-call dispatch with configurable concurrency (default 1, serial) and 2-second hold-filler delay default
- Panic recovery around `Observer` callbacks and `ToolExecutor.Execute`; `ErrToolExecutorPanicked` sentinel
- Sentinel: `ErrFormatBridge` for unsupported format pairs
- `fake/` subpackage: `Transport`, `LLM`, `Executor`, `Filler`, `RecorderObserver`
- 20+ unit tests (race-clean), 3 runnable examples, 1 fuzz target
- Statement coverage: `codec_bridge.go` 100%, `pipeline.go` ~85% (1 defensive guard unreachable without fake-transport/LLM error-injection helpers)

### Dependencies
- `github.com/mahdi-awadi/gopkg/audio/codec v0.1.0`
- `github.com/mahdi-awadi/gopkg/id v0.1.0`
- Go stdlib
