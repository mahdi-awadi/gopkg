package validate_test

import (
	"errors"
	"fmt"

	"github.com/mahdi-awadi/gopkg/validate"
)

func ExampleEmail() {
	fmt.Println(validate.Email("alice@example.com"))
	fmt.Println(errors.Is(validate.Email("not-an-email"), validate.ErrInvalidEmail))
	// Output:
	// <nil>
	// true
}

func ExamplePhoneE164() {
	fmt.Println(validate.PhoneE164("+12025550199"))
	fmt.Println(validate.PhoneE164("not a phone"))
	// Output:
	// <nil>
	// validate: invalid phone (E.164 required)
}

func ExamplePassword() {
	p := validate.DefaultPasswordPolicy()
	err := validate.Password("Strong1!pw", p)
	fmt.Println(err == nil)
	// Output: true
}

func ExampleNormalizeEmail() {
	fmt.Println(validate.NormalizeEmail("  ALICE@Example.COM  "))
	// Output: alice@example.com
}
