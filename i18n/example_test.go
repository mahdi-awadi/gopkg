package i18n_test

import (
	"fmt"

	"github.com/mahdi-awadi/gopkg/i18n"
)

func ExampleLocalizedString_Get() {
	name := i18n.LocalizedString{"en": "Hilton", "ar": "هيلتون"}

	fmt.Println(name.Get("ar"))
	fmt.Println(name.Get("ku")) // fallback to en
	fmt.Println(name.Get("xx")) // fallback to en
	// Output:
	// هيلتون
	// Hilton
	// Hilton
}

func ExampleLocalizedString_Value() {
	// Satisfies database/sql/driver.Valuer — writes JSONB to Postgres.
	name := i18n.LocalizedString{"en": "Hi"}
	v, _ := name.Value()
	fmt.Println(string(v.([]byte)))
	// Output: {"en":"Hi"}
}
