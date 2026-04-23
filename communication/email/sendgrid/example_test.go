package sendgrid_test

import (
	"context"
	"fmt"

	sg "github.com/mahdi-awadi/gopkg/communication/email/sendgrid"
	"github.com/mahdi-awadi/gopkg/communication/provider"
)

func ExampleProvider_Send() {
	p := sg.New(sg.Config{
		APIKey:    "SG.xxxxx",
		FromEmail: "no-reply@example.com",
		FromName:  "Example",
	}, nil)

	_, _ = p.Send(context.Background(), &provider.SendRequest{
		RecipientEmail: "bob@example.com",
		Subject:        "Hi",
		Body:           "Plain text body",
		HTMLBody:       "<p>HTML body</p>",
	})
}

func ExampleProvider_SendWithAttachments() {
	p := sg.New(sg.Config{APIKey: "SG.xxx", FromEmail: "from@example.com"}, nil)
	_, _ = p.SendWithAttachments(context.Background(),
		&provider.SendRequest{RecipientEmail: "bob@example.com", Subject: "Your invoice"},
		[]provider.Attachment{{Filename: "invoice.pdf", ContentType: "application/pdf", Content: []byte{}}},
	)
}

func ExampleMaskEmail() {
	fmt.Println(sg.MaskEmail("alice@example.com"))
	// Output: al***@example.com
}
