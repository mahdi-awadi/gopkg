package slicex_test

import (
	"fmt"

	"github.com/mahdi-awadi/gopkg/slicex"
)

func ExampleMap() {
	names := slicex.Map([]int{1, 2, 3}, func(v int) string {
		return fmt.Sprintf("id-%d", v)
	})
	fmt.Println(names)
	// Output: [id-1 id-2 id-3]
}

func ExampleFilter() {
	adults := slicex.Filter([]int{12, 18, 25, 7, 30}, func(age int) bool {
		return age >= 18
	})
	fmt.Println(adults)
	// Output: [18 25 30]
}

func ExampleReduce() {
	sum := slicex.Reduce([]int{1, 2, 3, 4, 5}, 0, func(acc, v int) int { return acc + v })
	fmt.Println(sum)
	// Output: 15
}

func ExampleUnique() {
	fmt.Println(slicex.Unique([]string{"a", "b", "a", "c"}))
	// Output: [a b c]
}

func ExampleChunk() {
	c := slicex.Chunk([]int{1, 2, 3, 4, 5}, 2)
	fmt.Println(c)
	// Output: [[1 2] [3 4] [5]]
}
