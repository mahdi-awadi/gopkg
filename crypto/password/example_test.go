package password_test

import (
	"errors"
	"fmt"

	"github.com/mahdi-awadi/gopkg/crypto/password"
)

func Example() {
	hash, _ := password.Hash("hunter2")

	// Later, verify a login attempt
	err := password.Verify(hash, "hunter2")
	fmt.Println(err == nil)

	err = password.Verify(hash, "wrong")
	fmt.Println(errors.Is(err, password.ErrMismatch))
	// Output:
	// true
	// true
}
