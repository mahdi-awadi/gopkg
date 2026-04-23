# communication/telegram

Telegram Bot adapter implementing `communication/provider.Provider`.

```
go get github.com/mahdi-awadi/gopkg/communication/telegram@latest
```

## Quickstart

```go
import (
    "context"
    tg "github.com/mahdi-awadi/gopkg/communication/telegram"
    "github.com/mahdi-awadi/gopkg/communication/provider"
)

p := tg.New(tg.Config{BotToken: "123:ABC..."}, nil)

resp, err := p.Send(context.Background(), &provider.SendRequest{
    RecipientTelegramChatID: "12345",
    Body:                    "Hello from gopkg",
})
// Use HTMLBody instead for HTML formatting (parse_mode=HTML).
```

## Notes

- Uses the Telegram Bot API over HTTPS — stdlib `net/http` only.
- Zero third-party deps.
- Override `APIBaseURL` in `Config` for testing against a fake server.
- `GetStatus` returns `StatusSent` (Telegram has no per-message status API).

## License

[MIT](../../LICENSE)
