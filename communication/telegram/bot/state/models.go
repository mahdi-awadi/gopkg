package state

// ConversationState tracks multi-step command flows.
type ConversationState struct {
	Command string            `json:"command"`
	Step    int               `json:"step"`
	Data    map[string]string `json:"data"`
}
