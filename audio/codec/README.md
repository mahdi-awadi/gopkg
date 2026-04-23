# audio/codec

[![Go Reference](https://pkg.go.dev/badge/github.com/mahdi-awadi/gopkg/audio/codec.svg)](https://pkg.go.dev/github.com/mahdi-awadi/gopkg/audio/codec)

Zero-dependency audio-codec primitives for realtime voice pipelines.

## What it does

Telephony platforms (WhatsApp Calling, Twilio voice, SIP/RTP) deliver audio as 8 kHz G.711 μ-law byte streams. Realtime LLM audio services (Google Gemini Live, OpenAI Realtime, Deepgram) speak 16 kHz or 24 kHz linear-PCM byte streams. This package bridges the two without pulling in a DSP library.

Every function is stateless, zero-allocation-predictable, and safe for concurrent use.

## Install

```bash
go get github.com/mahdi-awadi/gopkg/audio/codec
```

## Quickstart

```go
import "github.com/mahdi-awadi/gopkg/audio/codec"

// Inbound: telephony → LLM (e.g. audio from a phone call → STT input)
pcm16 := codec.Mulaw8kToPCM16LE16k(telephonyFrame)

// Outbound: LLM → telephony (e.g. TTS output → phone call)
mulaw := codec.PCM16LE24kToMulaw8k(llmFrame)
```

## API

| Function | Purpose |
|---|---|
| `MulawToPCM16(mulaw []byte) []int16` | G.711 μ-law byte → signed 16-bit PCM sample |
| `PCM16ToMulaw(pcm []int16) []byte` | Signed 16-bit PCM sample → G.711 μ-law byte |
| `Upsample8to16(samples []int16) []int16` | Double sample rate with linear interpolation |
| `Downsample24to8(samples []int16) []int16` | Drop to one-third rate via decimation |
| `Mulaw8kToPCM16LE16k(mulaw []byte) []byte` | Convenience: telephony → LLM-format bytes |
| `PCM16LE24kToMulaw8k(pcmData []byte) []byte` | Convenience: LLM-format bytes → telephony |

All helpers accept and return slices; lengths map one-to-one (codec) or by a fixed ratio (resample/bridge). See godoc for exact ratios.

## Dependencies

None beyond the Go standard library (`encoding/binary`).

## License

MIT © Mahdi Awadi
