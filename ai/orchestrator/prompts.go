package orchestrator

import "fmt"

// ChannelPrompt is a per-channel style hint.
type ChannelPrompt struct {
	Channel string
	Hint    string
}

// BuildSystemPrompt prepends a channel hint to a tenant persona.
func BuildSystemPrompt(persona string, hint ChannelPrompt) string {
	if hint.Hint == "" {
		return persona
	}
	return fmt.Sprintf("[Channel: %s - %s]\n\n%s", hint.Channel, hint.Hint, persona)
}

// DefaultChannelHints returns baseline style hints for supported channels.
func DefaultChannelHints() map[string]ChannelPrompt {
	return map[string]ChannelPrompt{
		"twilio:voice": {
			Channel: "twilio:voice",
			Hint:    "Speak naturally for a phone call. Short sentences, numbers spelled out, no bullet lists.",
		},
		"twilio:whatsapp": {
			Channel: "twilio:whatsapp",
			Hint:    "Reply in short paragraphs. WhatsApp bold uses *asterisks*.",
		},
		"meta:whatsapp": {
			Channel: "meta:whatsapp",
			Hint:    "Reply in short paragraphs. WhatsApp bold uses *asterisks*.",
		},
		"twilio:sms": {
			Channel: "twilio:sms",
			Hint:    "Keep replies under 160 characters when possible.",
		},
		"telegram": {
			Channel: "telegram",
			Hint:    "Markdown is supported. Use formatting when it improves clarity.",
		},
		"web": {
			Channel: "web",
			Hint:    "In-website chat can use concise Markdown and slightly longer replies.",
		},
		"email:sendgrid": {
			Channel: "email:sendgrid",
			Hint:    "Email reply. Use one to three short paragraphs.",
		},
		"email:ses": {
			Channel: "email:ses",
			Hint:    "Email reply. Use one to three short paragraphs.",
		},
	}
}
