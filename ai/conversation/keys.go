package conversation

import "fmt"

// KeyForUser builds the canonical Redis key for this user's AI
// conversation history across all channels.
func KeyForUser(partnerID, userID string) string {
	return fmt.Sprintf("ai:conversation:%s:%s", partnerID, userID)
}
