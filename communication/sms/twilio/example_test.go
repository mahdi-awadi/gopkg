package twilio_test

import (
	"context"
	"fmt"

	"github.com/mahdi-awadi/gopkg/communication/provider"
	twilsms "github.com/mahdi-awadi/gopkg/communication/sms/twilio"
)

func ExampleProvider_Send() {
	p := twilsms.New(twilsms.Config{
		AccountSID: "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		AuthToken:  "auth_token",
		FromNumber: "+15005550006",
	}, nil)

	// In production you'd capture resp, err
	_, _ = p.Send(context.Background(), &provider.SendRequest{
		RecipientPhone: "+12025550199",
		Body:           "Your verification code is 1234",
	})
}

func ExampleMaskPhone() {
	fmt.Println(twilsms.MaskPhone("+19175550100"))
	// Output: +1917555****
}
