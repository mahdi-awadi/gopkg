package environment_test

import (
	"fmt"

	"github.com/mahdi-awadi/gopkg/environment"
)

func Example() {
	// Typically called once in main() to gate dev-only behavior.
	if environment.IsDevelopment() {
		fmt.Println("enabling verbose logging")
	}
	_ = environment.GetEnvironment() // returns one of Development/Testing/Staging/Production
}
