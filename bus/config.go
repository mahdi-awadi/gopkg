package bus

// Config holds broker configuration.
//
// The library does not read environment variables; each consuming service
// constructs Config directly in its wiring (main.go / app package).
type Config struct {
	// Type identifies the broker backend. Currently only "nats" is
	// supported, but future adapters (e.g. Pub/Sub) will use this to
	// select the right constructor in the consumer's wiring code.
	Type string

	// URL is the connection string for the backend (e.g. "nats://host:4222").
	URL string

	// User, Password are the optional basic-auth credentials.
	User     string
	Password string

	// ServiceName is stamped into durable consumer/subscription names
	// and surfaced in logs/metrics for observability.
	ServiceName string
}
