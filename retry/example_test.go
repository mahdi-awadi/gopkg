package retry_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mahdi-awadi/gopkg/retry"
)

func ExampleDo() {
	calls := 0
	err := retry.Do(context.Background(),
		retry.Policy{MaxAttempts: 3, InitialDelay: time.Millisecond, Multiplier: 1},
		func() error {
			calls++
			if calls < 2 {
				return errors.New("try again")
			}
			return nil
		},
	)
	fmt.Println(calls, err)
	// Output: 2 <nil>
}

func ExamplePermanent() {
	calls := 0
	err := retry.Do(context.Background(),
		retry.Policy{MaxAttempts: 10, InitialDelay: time.Millisecond, Multiplier: 1},
		func() error {
			calls++
			return retry.Permanent(errors.New("403 forbidden — don't retry"))
		},
	)
	fmt.Println(calls, errors.Is(err, retry.ErrPermanent))
	// Output: 1 true
}
