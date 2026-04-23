package ratelimit_test

import (
	"fmt"
	"time"

	"github.com/mahdi-awadi/gopkg/ratelimit"
)

func ExampleLimiter_Allow() {
	// 5 ops / second, burst of 5.
	l := ratelimit.New(5, 5)

	allowed := 0
	for i := 0; i < 10; i++ {
		if l.Allow() {
			allowed++
		}
	}
	fmt.Println("allowed within burst:", allowed)
	// Output: allowed within burst: 5
}

func ExampleNewEvery() {
	// 10 ops, refilling at 1 per second.
	l := ratelimit.NewEvery(10, time.Second)
	fmt.Println(l.Allow())
	// Output: true
}
