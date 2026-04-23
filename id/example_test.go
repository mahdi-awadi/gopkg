package id_test

import (
	"fmt"
	"strings"

	"github.com/mahdi-awadi/gopkg/id"
)

func ExampleUUIDv7() {
	u := id.UUIDv7()
	// RFC 9562 v7: 36 chars, 4 hyphens, version nibble = '7' at position 14.
	fmt.Println(len(u))
	fmt.Println(strings.Count(u, "-"))
	fmt.Println(string(u[14]))
	// Output:
	// 36
	// 4
	// 7
}
