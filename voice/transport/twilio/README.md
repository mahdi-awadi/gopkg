# voice/transport/twilio

Twilio Programmable Voice Media Streams transport for `gopkg/voice/pipeline`.

```bash
go get github.com/mahdi-awadi/gopkg/voice/transport/twilio@latest
```

## Usage

```go
start, err := twilio.ReadStart(ctx, wsConn)
if err != nil {
    return err
}

transport := twilio.NewTransport(wsConn, start.StreamSid)
```

The package handles Twilio's WebSocket media protocol only. It does not own
TwiML routing, webhook signature validation, credentials, tenant lookup, prompts,
or tool execution.

## Security

No credentials are accepted or stored by this package. Applications remain
responsible for validating Twilio webhooks before upgrading or proxying WebSocket
traffic.

## License

MIT
