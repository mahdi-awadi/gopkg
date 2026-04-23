package signals_test

import (
	"context"
	"fmt"

	"github.com/mahdi-awadi/gopkg/signals"
)

func Example() {
	// In a real program this would wrap your server loop.
	ctx, cancel := signals.NotifyContext(context.Background())
	defer cancel()

	_ = ctx // pass into server.Run(ctx); cancelled on SIGINT/SIGTERM

	fmt.Println("waiting for shutdown signal")
	// Output: waiting for shutdown signal
}
