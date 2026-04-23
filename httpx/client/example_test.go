package client_test

import (
	"time"

	"github.com/mahdi-awadi/gopkg/httpx/client"
)

func ExampleNew() {
	c := client.New(client.Config{
		Timeout: 10 * time.Second,
		Retry: client.RetryConfig{
			MaxAttempts:    5,
			BackoffInitial: 200 * time.Millisecond,
			BackoffMax:     3 * time.Second,
			// Default retry statuses: 502, 503, 504. Override with RetryOnStatuses.
		},
	})
	_ = c // use c.Get / c.Post / c.Do
}
