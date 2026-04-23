package expo_test

import (
	"context"
	"fmt"

	"github.com/mahdi-awadi/gopkg/communication/provider"
	"github.com/mahdi-awadi/gopkg/communication/push/expo"
)

func ExampleProvider_Send() {
	p := expo.New(nil) // no creds required; nil logger → noop
	_, _ = p.Send(context.Background(), &provider.SendRequest{
		RecipientDeviceToken: "ExponentPushToken[xxxxxx]",
		Subject:              "Hello",
		Body:                 "Tap to view",
	})
}

func ExampleIsExpoToken() {
	fmt.Println(expo.IsExpoToken("ExponentPushToken[abc]"))
	fmt.Println(expo.IsExpoToken("fcm-token"))
	// Output:
	// true
	// false
}
