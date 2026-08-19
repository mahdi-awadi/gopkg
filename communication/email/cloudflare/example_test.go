package cloudflare_test

import (
	"context"
	"fmt"

	cf "github.com/mahdi-awadi/gopkg/communication/email/cloudflare"
	"github.com/mahdi-awadi/gopkg/communication/provider"
)

func ExampleProvider_Send() {
	p := cf.New(cf.Config{
		WorkerURL: "https://mail.example.workers.dev/send",
		AuthToken: "worker-secret",
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
	p := cf.New(cf.Config{
		WorkerURL: "https://mail.example.workers.dev/send",
		AuthToken: "worker-secret",
		FromEmail: "no-reply@example.com",
	}, nil)
	_, _ = p.SendWithAttachments(context.Background(),
		&provider.SendRequest{RecipientEmail: "bob@example.com", Subject: "Your invoice"},
		[]provider.Attachment{{Filename: "invoice.pdf", ContentType: "application/pdf", Content: []byte{}}},
	)
}

func ExampleMaskEmail() {
	fmt.Println(cf.MaskEmail("alice@example.com"))
	// Output: al***@example.com
}
