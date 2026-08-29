package gemini

import (
	"github.com/mahdi-awadi/gopkg/voice/pipeline"
)

// BuildSetup constructs the wire payload sent on Open.
//
// systemPrompt: caller-supplied system instruction (any language, any
// persona). Empty string is permitted but typically a mistake — the
// model will fall back to its baseline behaviour.
//
// tools: caller-supplied tool list. Pass nil or empty to disable tool
// calling entirely (the wire payload will omit the "tools" field).
func BuildSetup(systemPrompt string, tools []pipeline.ToolDecl) GeminiSetup {
	var wireTools []GeminiTool
	if len(tools) > 0 {
		wireTools = toolsToWire(tools)
	}
	return GeminiSetup{
		Setup: GeminiSetupConfig{
			Model: LiveModel,
			GenerationConfig: GeminiGenerationConfig{
				ResponseModalities: []string{"AUDIO"},
				SpeechConfig: &GeminiSpeechConfig{
					VoiceConfig: GeminiVoiceConfig{
						PrebuiltVoiceConfig: GeminiPrebuiltVoice{VoiceName: VoiceName},
					},
				},
			},
			SystemInstruction:       &GeminiContent{Parts: []GeminiPart{{Text: systemPrompt}}},
			Tools:                   wireTools,
			InputAudioTranscription: &struct{}{},
			// Phone-call VAD: low start sensitivity so ambient noise / speaker
			// echo does not interrupt the model mid-reply; a clear voice still
			// barges in. Longer silence window avoids clipping on brief pauses.
			RealtimeInputConfig: &GeminiRealtimeInputConfig{
				AutomaticActivityDetection: &GeminiAutomaticActivityDetection{
					StartOfSpeechSensitivity: "START_SENSITIVITY_LOW",
					EndOfSpeechSensitivity:   "END_SENSITIVITY_LOW",
					PrefixPaddingMs:          300,
					SilenceDurationMs:        600,
				},
			},
		},
	}
}

// toolsToWire converts pipeline.ToolDecl into the on-the-wire GeminiTool
// shape. Replaces an earlier helper that took a tenant-specific tool
// type; callers (eticket, Saffar, future tenants) now own their
// toolsets and this package stays domain-neutral.
func toolsToWire(in []pipeline.ToolDecl) []GeminiTool {
	decls := make([]GeminiFunctionDecl, 0, len(in))
	for _, t := range in {
		decls = append(decls, GeminiFunctionDecl{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  schemaToWire(t.Parameters),
		})
	}
	return []GeminiTool{{FunctionDeclarations: decls}}
}

// schemaToWire maps pipeline.ToolSchema → the map[string]any the
// Gemini Live wire format expects.
func schemaToWire(s pipeline.ToolSchema) map[string]any {
	props := map[string]any{}
	for name, p := range s.Properties {
		prop := map[string]any{
			"type":        p.Type,
			"description": p.Description,
		}
		if len(p.Enum) > 0 {
			prop["enum"] = p.Enum
		}
		if p.Format != "" {
			prop["format"] = p.Format
		}
		props[name] = prop
	}
	out := map[string]any{
		"type":       s.Type,
		"properties": props,
	}
	if len(s.Required) > 0 {
		out["required"] = s.Required
	}
	return out
}
