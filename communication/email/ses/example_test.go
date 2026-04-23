package ses_test

import (
	"context"
	"fmt"

	"github.com/mahdi-awadi/gopkg/communication/email/ses"
	"github.com/mahdi-awadi/gopkg/communication/provider"
)

func Example() {
	ctx := context.Background()
	p, err := ses.New(ctx, ses.Config{
		AccessKeyID:     "AKIA...",
		SecretAccessKey: "secret",
		Region:          "us-east-1",
		FromEmail:       "no-reply@example.com",
		FromName:        "Example",
	}, nil)
	if err != nil {
		return
	}
	_, _ = p.Send(ctx, &provider.SendRequest{
		RecipientEmail: "bob@example.com",
		Subject:        "Hello",
		Body:           "Plain text",
	})
}

func ExampleMaskEmail() {
	fmt.Println(ses.MaskEmail("a@example.com"))
	// Output: a***@example.com
}
