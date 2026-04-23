package clock_test

import (
	"fmt"
	"time"

	"github.com/mahdi-awadi/gopkg/clock"
)

func ExampleMock() {
	m := clock.NewMock(time.Date(2026, 4, 23, 0, 0, 0, 0, time.UTC))

	ch := m.After(5 * time.Second)

	// advance 5 seconds — channel fires
	m.Advance(5 * time.Second)
	<-ch

	fmt.Println(m.Now().Format(time.RFC3339))
	// Output: 2026-04-23T00:00:05Z
}
