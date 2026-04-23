package bus

import (
	"context"
)

// Message represents a received message from the broker.
// Implementations wrap an adapter-specific type and expose just enough
// to decode the payload. The byte slice returned by Data is read-only.
type Message interface {
	// Subject returns the subject (topic) the message was published to.
	Subject() string
	// Data returns the raw payload bytes. Do not mutate.
	Data() []byte
}

// MessageHandler processes a received message.
// Returning nil acknowledges the message (success).
// Returning a non-nil error negatively-acknowledges (adapter redelivery policy applies).
type MessageHandler func(ctx context.Context, msg Message) error

// PublishOption configures a single publish call.
type PublishOption func(*PublishConfig)

// PublishConfig is the resolved publish configuration. Adapters may read
// these fields directly; callers configure via the PublishOption helpers.
type PublishConfig struct {
	OrderingKey string
	Headers     map[string]string
}

// ApplyPublishOptions resolves a slice of PublishOption into a
// PublishConfig. Adapters should call this at the start of PublishRaw.
func ApplyPublishOptions(opts []PublishOption) PublishConfig {
	cfg := PublishConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// WithOrderingKey sets the ordering key for publishes.
// Used by Pub/Sub-style adapters for per-key ordering; ignored by NATS.
func WithOrderingKey(key string) PublishOption {
	return func(c *PublishConfig) {
		c.OrderingKey = key
	}
}

// WithHeaders merges metadata headers onto the message.
// Keys are adapter-specific (e.g. "Nats-Msg-Id" for JetStream idempotency).
// Call multiple times to accumulate.
func WithHeaders(h map[string]string) PublishOption {
	return func(c *PublishConfig) {
		if c.Headers == nil {
			c.Headers = make(map[string]string, len(h))
		}
		for k, v := range h {
			c.Headers[k] = v
		}
	}
}

// Broker is the full event broker interface.
//
// Broker implementations MUST be safe for concurrent use by multiple
// goroutines. Every method that performs I/O honors ctx cancellation.
type Broker interface {
	// PublishRaw publishes a pre-serialized payload (JSON, proto-bytes, …)
	// to the given subject. The library does not marshal on behalf of the
	// caller; encode your event to bytes (or to a type the adapter can
	// JSON-marshal) before calling.
	PublishRaw(ctx context.Context, subject string, data interface{}, opts ...PublishOption) error

	// Subscribe creates a durable subscription.
	// topic maps to NATS stream / Pub/Sub topic.
	// subscription maps to NATS consumer / Pub/Sub subscription.
	// filter maps to NATS filter subject / Pub/Sub filter.
	Subscribe(ctx context.Context, topic, subscription, filter string, handler MessageHandler) (Subscription, error)

	// Health checks the broker connection health.
	Health() error

	// Drain waits for in-flight messages before closing.
	Drain() error

	// Close closes the broker connection immediately.
	Close() error
}

// Subscription represents an active subscription that can be stopped.
type Subscription interface {
	Stop()
}

