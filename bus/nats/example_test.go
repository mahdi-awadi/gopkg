package nats_test

import (
	"context"

	"github.com/mahdi-awadi/gopkg/bus"
	busnats "github.com/mahdi-awadi/gopkg/bus/nats"
)

// Example shows how to construct a JetStream-backed Broker and perform
// basic publish + subscribe. It does not connect to a live NATS server
// — it's here for godoc rendering.
func Example() {
	ctx := context.Background()

	b, err := busnats.NewBroker(&bus.Config{
		URL:         "nats://localhost:4222",
		ServiceName: "orders",
	}, bus.NoopLogger{})
	if err != nil {
		return
	}
	defer b.Close()

	_ = b.PublishRaw(ctx, "orders.created", []byte(`{"id":"o_123"}`))

	sub, err := b.Subscribe(ctx, "ORDERS", "orders-worker", "orders.>", func(ctx context.Context, m bus.Message) error {
		// handle m.Subject(), m.Data()
		return nil
	})
	if err != nil {
		return
	}
	defer sub.Stop()
}
