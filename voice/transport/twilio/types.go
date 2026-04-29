// Package twilio implements pipeline.Transport over Twilio Programmable
// Voice Media Streams WebSockets.
package twilio

// Message is the union of inbound WebSocket messages Twilio sends:
// connected, start, media, mark, and stop.
type Message struct {
	Event          string `json:"event"`
	SequenceNumber string `json:"sequenceNumber,omitempty"`
	Protocol       string `json:"protocol,omitempty"`
	Version        string `json:"version,omitempty"`
	Start          *Start `json:"start,omitempty"`
	Media          *Media `json:"media,omitempty"`
	Mark           *Mark  `json:"mark,omitempty"`
	Stop           *Stop  `json:"stop,omitempty"`
	StreamSid      string `json:"streamSid,omitempty"`
}

// Start is Twilio's stream-start payload.
type Start struct {
	StreamSid        string            `json:"streamSid"`
	AccountSid       string            `json:"accountSid"`
	CallSid          string            `json:"callSid"`
	Tracks           []string          `json:"tracks"`
	MediaFormat      MediaFormat       `json:"mediaFormat"`
	CustomParameters map[string]string `json:"customParameters"`
}

// MediaFormat describes Twilio's inbound media format.
type MediaFormat struct {
	Encoding   string `json:"encoding"`
	SampleRate int    `json:"sampleRate"`
	Channels   int    `json:"channels"`
}

// Media carries one base64-encoded mu-law audio chunk.
type Media struct {
	Track     string `json:"track"`
	Chunk     string `json:"chunk"`
	Timestamp string `json:"timestamp"`
	Payload   string `json:"payload"`
}

// Mark is both Twilio's mark acknowledgement payload and the outbound mark
// payload accepted by Twilio.
type Mark struct {
	Name string `json:"name"`
}

// Stop signals that Twilio ended the media stream.
type Stop struct {
	AccountSid string `json:"accountSid"`
	CallSid    string `json:"callSid"`
}

// OutMessage is the outbound WebSocket envelope Twilio accepts.
type OutMessage struct {
	Event     string    `json:"event"`
	StreamSid string    `json:"streamSid"`
	Media     *OutMedia `json:"media,omitempty"`
	Mark      *Mark     `json:"mark,omitempty"`
}

// OutMedia carries one base64-encoded outbound audio chunk.
type OutMedia struct {
	Payload string `json:"payload"`
}
