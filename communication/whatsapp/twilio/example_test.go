package twilio_test

import (
	"context"
	"fmt"

	"github.com/mahdi-awadi/gopkg/communication/provider"
	tw "github.com/mahdi-awadi/gopkg/communication/whatsapp/twilio"
)

func ExampleProvider_Send() {
	p := tw.New(tw.Config{
		AccountSID: "ACxxxx",
		AuthToken:  "token",
		From:       "+15005550006",
	}, nil)
	_, _ = p.Send(context.Background(), &provider.SendRequest{
		RecipientPhone: "+12025550199",
		Body:           "Hi via WhatsApp",
	})
}

func ExampleProvider_SendTemplate() {
	p := tw.New(tw.Config{
		AccountSID: "ACxxxx",
		AuthToken:  "token",
		From:       "+15005550006",
		ContentSid: "HXxxxx...",
	}, nil)
	_, _ = p.SendTemplate(context.Background(),
		&provider.SendRequest{RecipientPhone: "+12025550199"},
		"ignored-by-twilio",
		[]string{"Alice", "1234"},
	)
}

func ExampleMaskPhone() {
	fmt.Println(tw.MaskPhone("whatsapp:+15551234"))
	// Output: +1555****
}
