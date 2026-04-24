# voice/pipeline

[![Go Reference](https://pkg.go.dev/badge/github.com/mahdi-awadi/gopkg/voice/pipeline.svg)](https://pkg.go.dev/github.com/mahdi-awadi/gopkg/voice/pipeline)

Realtime-LLM voice bridge. Plug a telephony `Transport` (Twilio Media Streams, WhatsApp Calling) into a realtime voice `LLM` (Gemini Live, OpenAI Realtime) with interruption handling, tool-call dispatch, session history, and pluggable hold-audio while tools are slow.

## Install

```bash
go get github.com/mahdi-awadi/gopkg/voice/pipeline
```

## 30-second quickstart

```go
p, _ := pipeline.New(pipeline.Options{
    Observer: myObserver,          // NoopObserver{} by default
    Filler:   myHoldToneFiller,    // SilentFiller{} by default
})

err := p.Run(ctx, transport, llm, toolExec,
    pipeline.SetupRequest{
        SystemPrompt: "You are a polite voice agent.",
        Tools:        []pipeline.ToolDecl{...},
        History:      []pipeline.HistoryTurn{...},
    },
    map[string]string{"user_id": "u-123"},
)
```

## Interfaces

| Interface | Role | Adapters |
|---|---|---|
| `Transport` | Caller-side endpoint (phone, WebRTC). | `voice/transport/twilio`, `voice/transport/whatsapp` |
| `LLM` | Realtime voice LLM session. | `voice/llm/gemini` |
| `ToolExecutor` | Runs LLM-initiated tool calls. | Consumer-supplied |
| `HoldFiller` | Audio while tools are slow. | `SilentFiller` default; custom for hold tone |
| `Observer` | Business-event callbacks (transcripts, tool calls, end). | `NoopObserver` default; `Multi(...)` composition |
| `Logger` | Structured logging. | `NoopLogger` default |

## Options

| Field | Default | Purpose |
|---|---|---|
| `ToolConcurrency` | `1` (serial) | Max parallel tool calls per LLM batch |
| `HoldFillerDelay` | `2s` | Grace before hold-filler starts |
| `Filler` | `SilentFiller{}` | Produces audio during slow tools |
| `Observer` | `NoopObserver{}` | Business-event callbacks |
| `Logger` | `NoopLogger{}` | Internal structured logs |
| `SessionIDFunc` | `gopkg/id.UUIDv7` | Generates each session's ID |

## Session termination

`Run` always fires `Observer.OnSessionEnd` before returning. `EndReason`:

- `transport_closed` — caller hung up
- `llm_closed` — LLM session ended cleanly
- `context_done` — caller cancelled ctx
- `fatal_error` — `Transport.Send`, `LLM.SendAudio`, `LLM.Open`, etc. returned an error (Run returns the wrapped error)

## Codec bridge

If `Transport` and `LLM` speak different audio formats, the pipeline inserts a conversion from `gopkg/audio/codec`. v0.1.0 supports:

- `mulaw @ 8 kHz ↔ pcm16le @ 16 kHz` (Twilio/WhatsApp voice ↔ Gemini input)
- `pcm16le @ 24 kHz → mulaw @ 8 kHz` (Gemini output → Twilio/WhatsApp voice)

Unknown pairs → `ErrFormatBridge` at `Run` start (no `OnSessionStart`).

## Testing

`fake/` subpackage ships scriptable fakes for every interface: `FakeTransport`, `FakeLLM`, `FakeExecutor`, `FakeFiller`, `RecorderObserver`. See `example_test.go` for usage.

## Dependencies

- `github.com/mahdi-awadi/gopkg/audio/codec`
- `github.com/mahdi-awadi/gopkg/id`
- Go stdlib only otherwise

## License

MIT © Mahdi Awadi
