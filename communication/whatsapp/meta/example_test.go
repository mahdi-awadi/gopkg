package meta_test

import (
	"context"
	"fmt"

	"github.com/mahdi-awadi/gopkg/communication/provider"
	"github.com/mahdi-awadi/gopkg/communication/whatsapp/meta"
)

func ExampleProvider_Send() {
	p := meta.New(meta.Config{
		PhoneNumberID: "1234567890",
		AccessToken:   "EAAG...",
	}, nil)
	_, _ = p.Send(context.Background(), &provider.SendRequest{
		RecipientPhone: "+12025550199",
		Body:           "Hello from gopkg",
	})
}

func ExampleProvider_SendTemplate() {
	p := meta.New(meta.Config{PhoneNumberID: "1", AccessToken: "t"}, nil)
	_, _ = p.SendTemplate(context.Background(),
		&provider.SendRequest{RecipientPhone: "+1...", Language: "en_US"},
		"order_confirmation",
		[]string{"Alice", "ORD-1234", "$42.00"},
	)
}

func ExampleMaskPhone() {
	fmt.Println(meta.MaskPhone("+12025551234"))
	// Output: 1202555****
}
