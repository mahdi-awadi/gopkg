// Package conversation is the shared conversation-history store for
// Gemini-backed assistants (voice + chat). It writes to Redis with a
// fixed TTL and rolling window so a user's history is consistent across
// channels.
package conversation
