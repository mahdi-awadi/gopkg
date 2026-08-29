// Package gemini implements pipeline.LLM over the Gemini Live
// WebSocket (v1beta BidiGenerateContent). Inbound LLM audio is pcm16le
// @ 24 kHz; outbound caller audio is pcm16le @ 16 kHz — the pipeline's
// codec bridge wires this to a Transport's native format (e.g. Twilio's
// mulaw@8k).
//
// The package is domain-neutral: callers supply the system prompt and
// tool list via pipeline.SetupRequest. An optional wake signal — a
// non-natural-language tag like "[CALL_CONNECTED]" — may be passed in
// SetupRequest.Extra["wake_signal"] (string) to make the model produce
// its first response without waiting for caller speech.
package gemini

// Wire-format constants. LiveModel + VoiceName match the Gemini Live API
// defaults used across all current tenants; override at the caller layer
// if you need a different model or voice in future.
const (
	LiveModel    = "models/gemini-3.1-flash-live-preview"
	LiveEndpoint = "wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1beta.GenerativeService.BidiGenerateContent"
	VoiceName    = "Zephyr"
)

// GeminiSetup is the top-level setup envelope.
type GeminiSetup struct {
	Setup GeminiSetupConfig `json:"setup"`
}

// GeminiSetupConfig is the setup message body — model, generation
// config, system prompt, tool decls, and (at this level, NOT under
// generationConfig) input audio transcription. The Gemini Live API
// rejects `setup.generationConfig.inputAudioTranscription` with
// "Cannot find field" — it's a top-level sibling of generationConfig.
type GeminiSetupConfig struct {
	Model                    string                 `json:"model"`
	GenerationConfig         GeminiGenerationConfig `json:"generationConfig"`
	SystemInstruction        *GeminiContent         `json:"systemInstruction,omitempty"`
	Tools                    []GeminiTool           `json:"tools,omitempty"`
	InputAudioTranscription  *struct{}                  `json:"inputAudioTranscription,omitempty"`
	OutputAudioTranscription *struct{}                  `json:"outputAudioTranscription,omitempty"`
	RealtimeInputConfig      *GeminiRealtimeInputConfig `json:"realtimeInputConfig,omitempty"`
}

// GeminiRealtimeInputConfig tunes how the Live API detects caller speech (voice
// activity detection). It is a top-level sibling of generationConfig. Important
// for phone calls: the default VAD is sensitive and interrupts the model's own
// speech on ambient noise or speaker-mode echo. Lower start sensitivity requires
// clearer speech before a barge-in fires.
type GeminiRealtimeInputConfig struct {
	AutomaticActivityDetection *GeminiAutomaticActivityDetection `json:"automaticActivityDetection,omitempty"`
}

// GeminiAutomaticActivityDetection configures Gemini Live's VAD. Sensitivity
// enums: START_SENSITIVITY_LOW/HIGH, END_SENSITIVITY_LOW/HIGH.
type GeminiAutomaticActivityDetection struct {
	Disabled                 bool   `json:"disabled,omitempty"`
	StartOfSpeechSensitivity string `json:"startOfSpeechSensitivity,omitempty"`
	EndOfSpeechSensitivity   string `json:"endOfSpeechSensitivity,omitempty"`
	PrefixPaddingMs          int    `json:"prefixPaddingMs,omitempty"`
	SilenceDurationMs        int    `json:"silenceDurationMs,omitempty"`
}

type GeminiGenerationConfig struct {
	ResponseModalities []string            `json:"responseModalities"`
	SpeechConfig       *GeminiSpeechConfig `json:"speechConfig,omitempty"`
}

type GeminiSpeechConfig struct {
	VoiceConfig  GeminiVoiceConfig `json:"voiceConfig"`
	LanguageCode string            `json:"languageCode,omitempty"`
}

type GeminiVoiceConfig struct {
	PrebuiltVoiceConfig GeminiPrebuiltVoice `json:"prebuiltVoiceConfig"`
}

type GeminiPrebuiltVoice struct {
	VoiceName string `json:"voiceName"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *GeminiInlineData `json:"inlineData,omitempty"`
}

type GeminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type GeminiTool struct {
	FunctionDeclarations []GeminiFunctionDecl `json:"functionDeclarations"`
}

type GeminiFunctionDecl struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// GeminiClientContent drives history replay and the initial wake
// signal (role="user", text=…).
type GeminiClientContent struct {
	ClientContent GeminiClientContentData `json:"clientContent"`
}

type GeminiClientContentData struct {
	TurnComplete bool         `json:"turnComplete"`
	Turns        []GeminiTurn `json:"turns"`
}

type GeminiTurn struct {
	Role  string       `json:"role"`
	Parts []GeminiPart `json:"parts"`
}

// GeminiRealtimeInput carries caller audio OR a text trigger.
type GeminiRealtimeInput struct {
	RealtimeInput GeminiRealtimeInputData `json:"realtimeInput"`
}

type GeminiRealtimeInputData struct {
	Audio *GeminiAudioData `json:"audio,omitempty"`
	Text  string           `json:"text,omitempty"`
}

type GeminiAudioData struct {
	Data     string `json:"data"`
	MimeType string `json:"mimeType"`
}

// GeminiServerMessage is the union of server-side messages.
type GeminiServerMessage struct {
	SetupComplete *struct{}            `json:"setupComplete,omitempty"`
	ServerContent *GeminiServerContent `json:"serverContent,omitempty"`
	ToolCall      *GeminiToolCall      `json:"toolCall,omitempty"`
}

type GeminiServerContent struct {
	ModelTurn           *GeminiContent       `json:"modelTurn,omitempty"`
	TurnComplete        bool                 `json:"turnComplete,omitempty"`
	Interrupted         bool                 `json:"interrupted,omitempty"`
	InputTranscription  *GeminiTranscription `json:"inputTranscription,omitempty"`
	OutputTranscription *GeminiTranscription `json:"outputTranscription,omitempty"`
}

type GeminiTranscription struct {
	Text string `json:"text"`
}

type GeminiToolCall struct {
	FunctionCalls []GeminiFunctionCall `json:"functionCalls"`
}

type GeminiFunctionCall struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type GeminiToolResponse struct {
	ToolResponse GeminiToolResponseData `json:"toolResponse"`
}

type GeminiToolResponseData struct {
	FunctionResponses []GeminiFunctionResponse `json:"functionResponses"`
}

type GeminiFunctionResponse struct {
	ID       string `json:"id"`
	Response any    `json:"response"`
}
