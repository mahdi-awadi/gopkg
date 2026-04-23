package mapx_test

import (
	"fmt"

	"github.com/mahdi-awadi/gopkg/mapx"
)

func ExampleKeysSorted() {
	m := map[string]int{"c": 3, "a": 1, "b": 2}
	fmt.Println(mapx.KeysSorted(m))
	// Output: [a b c]
}

func ExampleMerge() {
	base := map[string]int{"a": 1, "b": 2}
	overlay := map[string]int{"b": 20, "c": 3}
	fmt.Println(mapx.Merge(base, overlay))
	// Output: map[a:1 b:20 c:3]
}

func ExampleInvert() {
	fmt.Println(mapx.Invert(map[int]string{1: "a", 2: "b"}))
	// Output: map[a:1 b:2]
}
