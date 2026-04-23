package fcm_test

import (
	"context"
	"fmt"

	"github.com/mahdi-awadi/gopkg/communication/provider"
	"github.com/mahdi-awadi/gopkg/communication/push/fcm"
)

func Example() {
	ctx := context.Background()
	p, err := fcm.New(ctx, fcm.Config{
		ProjectID:       "my-firebase-project",
		CredentialsJSON: `{"type":"service_account", "..."}`,
	}, nil)
	if err != nil {
		return
	}
	_, _ = p.Send(ctx, &provider.SendRequest{
		RecipientDeviceToken: "fcm-token-123",
		Subject:              "New message",
		Body:                 "Tap to view",
	})
}

func ExampleMaskToken() {
	fmt.Println(fcm.MaskToken("aabbccddeeff11223344"))
	// Output: aabbccdd****
}
