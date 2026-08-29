// Package whatsapptransport implements pipeline.Transport over a pion WebRTC
// PeerConnection for WhatsApp Calling (WebRTC media over the WhatsApp Cloud API).
//
// # Overview
//
// When Adminli's comms service receives an inbound WhatsApp call, it obtains
// the caller's SDP offer from the WhatsApp Cloud API, constructs a Peer via
// NewPeer, and hands the resulting *Peer to New to obtain a pipeline.Transport.
// The pipeline runs caller audio (inbound) through an LLM (Gemini Live) and
// streams the LLM's response back to the caller (outbound).
//
// # Codec
//
// Only Opus (48000 Hz, 2-channel, RTP payload type 111) is negotiated — the
// codec advertised by the real WhatsApp Cloud API SDP offer. Transcoding is
// performed by CGO libopus (github.com/hraban/opus; requires libopus ≥ 1.1):
//
//   - Inbound (caller → Gemini): each Opus frame is decoded to PCM16LE @ 16 kHz
//     mono (libopus resamples internally). InboundFormat reports:
//     pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 16000, Channels: 1}
//
//   - Outbound (Gemini → caller): PCM16LE @ 24 kHz mono is buffered and encoded
//     in fixed 20 ms / 480-sample Opus frames. OutboundFormat reports:
//     pipeline.AudioFormat{Encoding: pipeline.EncodingPCM16LE, SampleRate: 24000, Channels: 1}
//
// # Build requirement
//
// This package uses CGO and requires libopus to be present:
//
//	apt-get install libopus-dev   # Debian/Ubuntu
//
// Build with CGO_ENABLED=1 (the default when a C toolchain is available).
//
// # Concurrency
//
// Only Send and Close acquire writeMu to serialise concurrent callers (Clear
// and Mark are lock-free no-ops). The pipeline drives Send from the LLM-events
// goroutine and may call Clear or Mark concurrently from the hold-filler pump
// goroutine.
package whatsapptransport
