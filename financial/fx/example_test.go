package fx_test

import (
	"context"
	"fmt"

	"github.com/mahdi-awadi/gopkg/financial/fx"
)

func Example() {
	rates := fx.NewMemory()
	_ = rates.SetPair("USD", "IQD", 1500)

	// Forward: 10 USD → IQD
	got, _ := rates.Convert(context.Background(), 10, "USD", "IQD")
	fmt.Println(got)

	// Backward: 3000 IQD → USD
	got, _ = rates.Convert(context.Background(), 3000, "IQD", "USD")
	fmt.Println(got)

	// Same currency always returns 1.0 rate.
	r, _ := rates.GetRate(context.Background(), "USD", "USD")
	fmt.Println(r)

	// Output:
	// 15000
	// 2
	// 1
}
