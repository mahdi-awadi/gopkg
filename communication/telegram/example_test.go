package telegram_test

import (
	"context"

	"github.com/mahdi-awadi/gopkg/communication/provider"
	"github.com/mahdi-awadi/gopkg/communication/telegram"
)

func Example() {
	p := telegram.New(telegram.Config{BotToken: "123456:ABC-DEF..."}, nil)
	_, _ = p.Send(context.Background(), &provider.SendRequest{
		RecipientTelegramChatID: "12345",
		Body:                    "Hello from gopkg",
	})
}
