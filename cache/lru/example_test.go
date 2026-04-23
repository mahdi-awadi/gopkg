package lru_test

import (
	"fmt"
	"time"

	"github.com/mahdi-awadi/gopkg/cache/lru"
)

func Example() {
	c := lru.New[string, int](100, 10*time.Minute)
	c.Set("user:42", 7)
	if v, ok := c.Get("user:42"); ok {
		fmt.Println(v)
	}
	// Output: 7
}
