# Changelog

## [unreleased] — voice/llm/gemini

### Breaking
- `SetupRequest.Extra["greeting_trigger"]` is no longer read. Callers
  that relied on the realtimeInput-text trigger pattern should
  migrate to `Extra["wake_signal"]` (sent as clientContent instead).
  Fixes saffar's 2026-05-13 greeting role-inversion bug where Gemini
  interpreted the greeting text as user speech and role-played as a
  customer.

### Added
- `SetupRequest.Extra["wake_signal"]` (string): when set, gemini's
  Open() sends one clientContent user turn with the wake text right
  after history replay. Use a bracketed tag (e.g. "[CALL_CONNECTED]")
  to avoid role inversion.

## [0.1.0] - 2026-04-28

### Added
- Gemini Live `pipeline.LLM` adapter.
- Configurable setup with model, voice, system prompt, tools, history, locale, and greeting text.
- Safe URL builder plus redaction helper for API-key-bearing URLs.
- Event translation for audio, assistant text, caller transcript, interruption, turn complete, and tool calls.
