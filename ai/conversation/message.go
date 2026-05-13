package conversation

import "time"

// Message is one turn in the conversation.
type Message struct {
	Role           string        `json:"role"`    // "user" | "model"
	Channel        string        `json:"channel"` // "voice" | "chat" | "telegram"
	Content        string        `json:"content"`
	FunctionCall   *FunctionCall `json:"function_call,omitempty"`
	DetectedIntent string        `json:"detected_intent,omitempty"`
	Timestamp      time.Time     `json:"timestamp"`
}

// FunctionCall captures a model-emitted tool call so it can be replayed
// on a later session to keep the model's context consistent.
type FunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
	ID   string         `json:"id,omitempty"`
}
