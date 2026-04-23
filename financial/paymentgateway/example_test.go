package paymentgateway_test

import (
	"fmt"

	"github.com/mahdi-awadi/gopkg/financial/paymentgateway"
)

func ExampleBuildCallbackURLWithReference() {
	built := paymentgateway.BuildCallbackURLWithReference(
		"https://api.example.com/payments/callback?tenant=7",
		"order-42",
	)
	fmt.Println(built)

	ref, _ := paymentgateway.ExtractReferenceFromPath(built)
	fmt.Println(ref)

	// Output:
	// https://api.example.com/payments/callback/order-42?tenant=7
	// order-42
}
