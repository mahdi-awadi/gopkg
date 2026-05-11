# voice/llm/gemini

Gemini Live WebSocket adapter for `gopkg/voice/pipeline`.

```bash
go get github.com/mahdi-awadi/gopkg/voice/llm/gemini@latest
```

## Usage

```go
conn, err := gemini.Dial(ctx, gemini.Config{
    APIKey: os.Getenv("GEMINI_API_KEY"),
})
if err != nil {
    return err
}
defer conn.Close()

llm := gemini.NewLLM(conn, gemini.Options{
    SystemPrompt: "You are a concise voice assistant.",
})
```

Applications provide prompts, tools, history, and credentials. This package
does not include application-specific prompts, tenant lookup, auth, storage, or
tool execution.

## Security

`BuildURL` returns a URL containing the API key. Never log it directly. Use
`RedactURL` for logs and error messages.

## License

MIT
