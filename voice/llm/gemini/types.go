// Package gemini implements pipeline.LLM over the Gemini Live WebSocket API.
package gemini

const (
	// DefaultModel is the model used when neither Options.Model nor
	// SetupRequest.Extra["model"] is provided.
	DefaultModel = "models/gemini-3.1-flash-live-preview"
	// DefaultEndpoint is Gemini Live's v1beta WebSocket endpoint.
	DefaultEndpoint = "wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1beta.GenerativeService.BidiGenerateContent"
	// DefaultVoiceName is used when no voice is supplied.
	DefaultVoiceName = "Zephyr"
)

// Setup is the top-level setup envelope sent to Gemini Live.
type Setup struct {
	Setup SetupConfig `json:"setup"`
}

// SetupConfig is Gemini Live's setup body.
type SetupConfig struct {
	Model                   string           `json:"model"`
	GenerationConfig        GenerationConfig `json:"generationConfig"`
	SystemInstruction       *Content         `json:"systemInstruction,omitempty"`
	Tools                   []Tool           `json:"tools,omitempty"`
	InputAudioTranscription *struct{}        `json:"inputAudioTranscription,omitempty"`
}

type GenerationConfig struct {
	ResponseModalities []string      `json:"responseModalities"`
	SpeechConfig       *SpeechConfig `json:"speechConfig,omitempty"`
}

type SpeechConfig struct {
	VoiceConfig  VoiceConfig `json:"voiceConfig"`
	LanguageCode string      `json:"languageCode,omitempty"`
}

type VoiceConfig struct {
	PrebuiltVoiceConfig PrebuiltVoice `json:"prebuiltVoiceConfig"`
}

type PrebuiltVoice struct {
	VoiceName string `json:"voiceName"`
}

type Content struct {
	Parts []Part `json:"parts"`
}

type Part struct {
	Text       string      `json:"text,omitempty"`
	InlineData *InlineData `json:"inlineData,omitempty"`
}

type InlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type Tool struct {
	FunctionDeclarations []FunctionDecl `json:"functionDeclarations"`
}

type FunctionDecl struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type ClientContent struct {
	ClientContent ClientContentData `json:"clientContent"`
}

type ClientContentData struct {
	TurnComplete bool   `json:"turnComplete"`
	Turns        []Turn `json:"turns"`
}

type Turn struct {
	Role  string `json:"role"`
	Parts []Part `json:"parts"`
}

type RealtimeInput struct {
	RealtimeInput RealtimeInputData `json:"realtimeInput"`
}

type RealtimeInputData struct {
	Audio *AudioData `json:"audio,omitempty"`
	Text  string     `json:"text,omitempty"`
}

type AudioData struct {
	Data     string `json:"data"`
	MimeType string `json:"mimeType"`
}

type ServerMessage struct {
	SetupComplete *struct{}      `json:"setupComplete,omitempty"`
	ServerContent *ServerContent `json:"serverContent,omitempty"`
	ToolCall      *ToolCall      `json:"toolCall,omitempty"`
}

type ServerContent struct {
	ModelTurn          *Content       `json:"modelTurn,omitempty"`
	TurnComplete       bool           `json:"turnComplete,omitempty"`
	Interrupted        bool           `json:"interrupted,omitempty"`
	InputTranscription *Transcription `json:"inputTranscription,omitempty"`
}

type Transcription struct {
	Text string `json:"text"`
}

type ToolCall struct {
	FunctionCalls []FunctionCall `json:"functionCalls"`
}

type FunctionCall struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type ToolResponse struct {
	ToolResponse ToolResponseData `json:"toolResponse"`
}

type ToolResponseData struct {
	FunctionResponses []FunctionResponse `json:"functionResponses"`
}

type FunctionResponse struct {
	ID       string `json:"id"`
	Response any    `json:"response"`
}
