package bus_test

import (
	"fmt"

	"github.com/mahdi-awadi/gopkg/bus"
)

// ExamplePublishOption demonstrates the functional-options pattern.
func ExamplePublishOption() {
	cfg := bus.ApplyPublishOptions([]bus.PublishOption{
		bus.WithOrderingKey("tenant-42"),
		bus.WithHeaders(map[string]string{"correlation_id": "abc-123"}),
	})
	fmt.Println(cfg.OrderingKey)
	fmt.Println(cfg.Headers["correlation_id"])
	// Output:
	// tenant-42
	// abc-123
}

// ExampleNoopLogger shows the zero-value logger you can inject anywhere
// a bus.Logger is accepted. It silently discards all log calls.
func ExampleNoopLogger() {
	var l bus.Logger = bus.NoopLogger{}
	l.Info("this is ignored", map[string]any{"reason": "noop"})
	l.Error("this too", nil)
	fmt.Println("done")
	// Output: done
}
