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
// Only PCMU (G.711 µ-law, 8000 Hz, mono, RTP payload type 0) is negotiated.
// PCMU RTP payload bytes are raw µ-law samples — no transcoding is needed inside
// this transport. Both InboundFormat and OutboundFormat report:
//
//	pipeline.AudioFormat{Encoding: pipeline.EncodingMulaw, SampleRate: 8000, Channels: 1}
//
// This matches the pipeline's built-in codec bridge.
//
// # Concurrency
//
// Send, Clear, Mark, and Close acquire writeMu to serialise concurrent callers
// (the pipeline drives Send from the LLM-events goroutine and may call Clear or
// Mark from the hold-filler pump goroutine concurrently).
package whatsapptransport
