package circuitbreaker_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/mahdi-awadi/gopkg/circuitbreaker"
)

func Example() {
	cfg := circuitbreaker.DefaultConfig()
	cfg.FailureThreshold = 2
	b := circuitbreaker.New("demo", cfg, nil)

	// Simulate 2 failures → breaker trips.
	for i := 0; i < 2; i++ {
		_ = b.Execute(context.Background(), func(context.Context) error {
			return errors.New("connection refused")
		}, nil)
	}

	// Further calls are rejected fast without running fn.
	err := b.Execute(context.Background(), func(context.Context) error {
		return nil // would succeed, but we never get here
	}, nil)
	fmt.Println(errors.Is(err, circuitbreaker.ErrCircuitOpen))
	fmt.Println(b.State())
	// Output:
	// true
	// open
}
