package gemini

import (
	"fmt"

	"github.com/mahdi-awadi/gopkg/voice/pipeline"
)

func buildSetup(req pipeline.SetupRequest, opts Options) Setup {
	model := firstNonEmpty(asString(req.Extra["model"]), opts.Model, DefaultModel)
	voice := firstNonEmpty(req.Voice, opts.VoiceName, DefaultVoiceName)
	prompt := firstNonEmpty(req.SystemPrompt, opts.SystemPrompt)
	locale := firstNonEmpty(req.LocaleHint, opts.LanguageCode)

	cfg := SetupConfig{
		Model: model,
		GenerationConfig: GenerationConfig{
			ResponseModalities: []string{"AUDIO"},
			SpeechConfig: &SpeechConfig{
				LanguageCode: locale,
				VoiceConfig: VoiceConfig{
					PrebuiltVoiceConfig: PrebuiltVoice{VoiceName: voice},
				},
			},
		},
		Tools: toWireTools(req.Tools),
	}
	if prompt != "" {
		cfg.SystemInstruction = &Content{Parts: []Part{{Text: prompt}}}
	}
	if opts.EnableInputTranscription {
		cfg.InputAudioTranscription = &struct{}{}
	}
	return Setup{Setup: cfg}
}

func toWireTools(in []pipeline.ToolDecl) []Tool {
	if len(in) == 0 {
		return nil
	}
	decls := make([]FunctionDecl, 0, len(in))
	for _, t := range in {
		decls = append(decls, FunctionDecl{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  convertSchema(t.Parameters),
		})
	}
	return []Tool{{FunctionDeclarations: decls}}
}

func convertSchema(s pipeline.ToolSchema) map[string]any {
	props := make(map[string]any, len(s.Properties))
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
		"type":       firstNonEmpty(s.Type, "object"),
		"properties": props,
	}
	if len(s.Required) > 0 {
		out["required"] = s.Required
	}
	return out
}

func historyMessage(turn pipeline.HistoryTurn) ClientContent {
	role := "user"
	if turn.Role == pipeline.RoleAssistant {
		role = "model"
	}
	return ClientContent{ClientContent: ClientContentData{
		TurnComplete: true,
		Turns: []Turn{{
			Role:  role,
			Parts: []Part{{Text: turn.Content}},
		}},
	}}
}

func greetingText(req pipeline.SetupRequest, opts Options) string {
	if v, ok := req.Extra["greeting_text"].(string); ok && v != "" {
		return v
	}
	if opts.GreetingText != "" {
		return opts.GreetingText
	}
	return "Hello"
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func missingSetupComplete(resp ServerMessage) error {
	return fmt.Errorf("gemini live: expected setupComplete, got: %+v", resp)
}
