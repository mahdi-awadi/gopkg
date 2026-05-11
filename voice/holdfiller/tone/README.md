# voice/holdfiller/tone

Mu-law hold-tone filler for `gopkg/voice/pipeline`.

```bash
go get github.com/mahdi-awadi/gopkg/voice/holdfiller/tone@latest
```

## Usage

```go
p, _ := pipeline.New(pipeline.Options{
    Filler: tone.New(),
})
```

The default emits a 2-second repeating pattern as 20ms mu-law frames: 400ms
soft tone followed by silence.

## License

MIT
