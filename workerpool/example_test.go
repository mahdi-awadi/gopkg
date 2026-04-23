package workerpool_test

import (
	"fmt"
	"sync/atomic"

	"github.com/mahdi-awadi/gopkg/workerpool"
)

func Example() {
	p := workerpool.New(8) // 8 concurrent workers
	var count int64
	for i := 0; i < 1000; i++ {
		p.Submit(func() { atomic.AddInt64(&count, 1) })
	}
	p.Wait()
	fmt.Println(count)
	// Output: 1000
}
