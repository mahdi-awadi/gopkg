package ptr_test

import (
	"fmt"

	"github.com/mahdi-awadi/gopkg/ptr"
)

type User struct {
	Name  string
	Email *string
}

func ExampleTo() {
	u := User{Name: "Alice", Email: ptr.To("alice@example.com")}
	fmt.Println(*u.Email)
	// Output: alice@example.com
}

func ExampleDeref() {
	var emailPtr *string
	fmt.Printf("%q\n", ptr.Deref(emailPtr)) // nil → ""
	s := "x"
	fmt.Printf("%q\n", ptr.Deref(&s))
	// Output:
	// ""
	// "x"
}

func ExampleOr() {
	var emailPtr *string
	fmt.Println(ptr.Or(emailPtr, "unknown@example.com"))
	// Output: unknown@example.com
}
