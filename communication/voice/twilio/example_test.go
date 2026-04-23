package twilio_test

import (
	"context"
	"fmt"

	"github.com/mahdi-awadi/gopkg/communication/provider"
	tv "github.com/mahdi-awadi/gopkg/communication/voice/twilio"
)

func ExampleProvider_Send() {
	p := tv.New(tv.Config{
		AccountSID: "ACxxxx",
		AuthToken:  "token",
		FromNumber: "+15005550006",
	}, nil)
	_, _ = p.Send(context.Background(), &provider.SendRequest{
		RecipientPhone: "+12025550199",
		// Body is intentionally empty — uses the default TwiML reject URL to
		// ring once and drop the call (flash-call verification).
	})
}

func ExampleMaskPhone() {
	fmt.Println(tv.MaskPhone("+15551234"))
	// Output: +1555****
}
