# Changelog

## [0.1.0] - 2026-04-23

### Added
- `MulawToPCM16(mulaw []byte) []int16` — G.711 μ-law decode (ITU-T 1972 table)
- `PCM16ToMulaw(pcm []int16) []byte` — G.711 μ-law encode with saturation clipping
- `Upsample8to16(samples []int16) []int16` — linear-interpolation 2× upsampling
- `Downsample24to8(samples []int16) []int16` — factor-3 decimation (no anti-alias filter)
- `Mulaw8kToPCM16LE16k(mulaw []byte) []byte` — telephony-format → LLM-format byte bridge
- `PCM16LE24kToMulaw8k(pcmData []byte) []byte` — LLM-format → telephony-format byte bridge
- 11 tests covering silence, max positive/negative, roundtrip, clipping, resampling edges, endianness
- 2 runnable examples
- Zero third-party dependencies
