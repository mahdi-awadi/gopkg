// Package pipeline bridges a telephony Transport (Twilio Media Streams,
// WhatsApp Calling, …) to a realtime voice LLM (Gemini Live, OpenAI
// Realtime, …), orchestrating audio in/out, tool-call dispatch,
// interruption, session history, and pluggable hold-audio while tools
// are slow.
//
// See README.md for quickstart; the public surface is:
//   - Pipeline, Options, New, Run
//   - Interfaces: Transport, LLM, ToolExecutor, HoldFiller, Observer, Logger
//   - Types: Frame, AudioFormat, Session, ToolCall, ToolResult, SetupRequest, …
//
// Adapters (voice/llm/gemini, voice/transport/twilio, voice/transport/whatsapp)
// live in sibling modules and implement this package's interfaces.
package pipeline
