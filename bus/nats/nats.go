// Package nats implements the bus.Broker interface using NATS JetStream.
//
// Self-contained: talks to github.com/nats-io/nats.go and jetstream directly,
// no dependency on any wrapper.
package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mahdi-awadi/gopkg/bus"
	natsgo "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Broker implements bus.Broker using NATS JetStream.
// Instances must be constructed via NewBroker; the zero value is not valid.
// Broker is safe for concurrent use.
type Broker struct {
	conn   *natsgo.Conn
	js     jetstream.JetStream
	logger bus.Logger
	mu     sync.RWMutex
	closed bool
}

// Compile-time check that *Broker satisfies bus.Broker.
var _ bus.Broker = (*Broker)(nil)

// NewBroker constructs a NATS JetStream-backed bus.Broker.
// The logger is optional; nil becomes bus.NoopLogger{}.
//
// Required: cfg.URL, cfg.ServiceName.
func NewBroker(cfg *bus.Config, logger bus.Logger) (bus.Broker, error) {
	if cfg == nil || cfg.URL == "" || cfg.ServiceName == "" {
		return nil, fmt.Errorf("nats: URL and ServiceName are required")
	}
	if logger == nil {
		logger = bus.NoopLogger{}
	}

	opts := []natsgo.Option{
		natsgo.Name(cfg.ServiceName),
		natsgo.ReconnectWait(2 * time.Second),
		natsgo.MaxReconnects(-1),
		natsgo.ReconnectHandler(func(nc *natsgo.Conn) {
			logger.Info("nats: reconnected", map[string]any{"url": nc.ConnectedUrl()})
		}),
		natsgo.DisconnectErrHandler(func(nc *natsgo.Conn, err error) {
			if err != nil {
				logger.Error("nats: disconnected", map[string]any{"error": err.Error()})
			}
		}),
	}
	if cfg.User != "" && cfg.Password != "" {
		opts = append(opts, natsgo.UserInfo(cfg.User, cfg.Password))
	}

	conn, err := natsgo.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("nats: connect %s: %w", cfg.URL, err)
	}
	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("nats: jetstream: %w", err)
	}

	logger.Info("nats: connected", map[string]any{
		"service": cfg.ServiceName,
		"url":     cfg.URL,
	})

	return &Broker{conn: conn, js: js, logger: logger}, nil
}

// PublishRaw serializes data to JSON and publishes via JetStream.
func (b *Broker) PublishRaw(ctx context.Context, subject string, data interface{}, opts ...bus.PublishOption) error {
	b.mu.RLock()
	closed := b.closed
	b.mu.RUnlock()
	if closed {
		return fmt.Errorf("nats: broker closed")
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("nats: marshal: %w", err)
	}
	if _, err := b.js.Publish(ctx, subject, bytes); err != nil {
		return fmt.Errorf("nats: publish %s: %w", subject, err)
	}
	return nil
}

// Subscribe creates a durable JetStream consumer and begins consuming.
func (b *Broker) Subscribe(ctx context.Context, stream, consumerName, filterSubject string, handler bus.MessageHandler) (bus.Subscription, error) {
	consumer, err := b.js.CreateOrUpdateConsumer(ctx, stream, jetstream.ConsumerConfig{
		Name:          consumerName,
		Durable:       consumerName,
		FilterSubject: filterSubject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		MaxDeliver:    5,
		DeliverPolicy: jetstream.DeliverNewPolicy,
	})
	if err != nil {
		return nil, fmt.Errorf("nats: create consumer %s: %w", consumerName, err)
	}

	consumeCtx, err := consumer.Consume(func(msg jetstream.Msg) {
		if err := handler(ctx, &natsMessage{msg: msg}); err != nil {
			b.logger.Error("nats: handler error", map[string]any{
				"subject": msg.Subject(),
				"error":   err.Error(),
			})
			_ = msg.Nak()
			return
		}
		_ = msg.Ack()
	})
	if err != nil {
		return nil, fmt.Errorf("nats: consume: %w", err)
	}

	b.logger.Info("nats: subscribed", map[string]any{
		"stream":   stream,
		"consumer": consumerName,
		"filter":   filterSubject,
	})

	return &natsSubscription{consumeCtx: consumeCtx}, nil
}

// Health returns nil if the NATS connection is in the connected state.
func (b *Broker) Health() error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.conn == nil || !b.conn.IsConnected() {
		return fmt.Errorf("nats: not connected")
	}
	return nil
}

// Drain drains in-flight publishes then closes the connection.
func (b *Broker) Drain() error {
	b.mu.RLock()
	conn := b.conn
	b.mu.RUnlock()
	if conn == nil {
		return nil
	}
	return conn.Drain()
}

// Close closes the NATS connection. Idempotent.
func (b *Broker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed || b.conn == nil {
		b.closed = true
		return nil
	}
	b.conn.Close()
	b.closed = true
	b.logger.Info("nats: closed", nil)
	return nil
}

// natsSubscription wraps jetstream.ConsumeContext as bus.Subscription.
type natsSubscription struct {
	consumeCtx jetstream.ConsumeContext
}

// Stop unsubscribes and releases resources. Idempotent.
func (s *natsSubscription) Stop() {
	if s.consumeCtx != nil {
		s.consumeCtx.Stop()
	}
}
