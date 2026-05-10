// Package twiliotransport implements pipeline.Transport over a Twilio
// Media Streams WebSocket. Twilio drives caller audio into the pipeline
// (via Receive) and consumes AI audio / Mark / Clear events (via Send,
// Mark, Clear).
//
// Frame format: mulaw, 8 kHz, 1 channel — Twilio's fixed wire format
// for Programmable Voice. The pipeline's codec bridge converts on the
// LLM side (pcm16@16k in, pcm16@24k out).
package twiliotransport

// TwilioMessage is the union of inbound WS messages Twilio sends:
// connected, start, media, mark (ack), stop.
type TwilioMessage struct {
	Event          string       `json:"event"`
	SequenceNumber string       `json:"sequenceNumber,omitempty"`
	Protocol       string       `json:"protocol,omitempty"`
	Version        string       `json:"version,omitempty"`
	Start          *TwilioStart `json:"start,omitempty"`
	Media          *TwilioMedia `json:"media,omitempty"`
	Mark           *TwilioMark  `json:"mark,omitempty"`
	Stop           *TwilioStop  `json:"stop,omitempty"`
	StreamSid      string       `json:"streamSid,omitempty"`
}

// TwilioStart is the initial handshake payload — CallSid, StreamSid,
// media format, and custom parameters from the TwiML <Stream> element.
type TwilioStart struct {
	StreamSid        string            `json:"streamSid"`
	AccountSid       string            `json:"accountSid"`
	CallSid          string            `json:"callSid"`
	Tracks           []string          `json:"tracks"`
	MediaFormat      TwilioMediaFormat `json:"mediaFormat"`
	CustomParameters map[string]string `json:"customParameters"`
}

// TwilioMediaFormat describes the wire encoding / sample rate /
// channel count Twilio is delivering.
type TwilioMediaFormat struct {
	Encoding   string `json:"encoding"`
	SampleRate int    `json:"sampleRate"`
	Channels   int    `json:"channels"`
}

// TwilioMedia carries one base64-encoded audio chunk (mulaw, 20ms).
type TwilioMedia struct {
	Track     string `json:"track"`
	Chunk     string `json:"chunk"`
	Timestamp string `json:"timestamp"`
	Payload   string `json:"payload"`
}

// TwilioMark is the ack Twilio sends after a mark event has played
// through the caller's device.
type TwilioMark struct {
	Name string `json:"name"`
}

// TwilioStop signals the caller hung up.
type TwilioStop struct {
	AccountSid string `json:"accountSid"`
	CallSid    string `json:"callSid"`
}

// TwilioOutMessage is the outbound WS message envelope Twilio accepts:
// media, mark, clear. StreamSid is required on every outbound message.
type TwilioOutMessage struct {
	Event     string          `json:"event"`
	StreamSid string          `json:"streamSid"`
	Media     *TwilioOutMedia `json:"media,omitempty"`
	Mark      *TwilioMark     `json:"mark,omitempty"`
}

// TwilioOutMedia carries one base64-encoded outbound audio chunk.
type TwilioOutMedia struct {
	Payload string `json:"payload"`
}
