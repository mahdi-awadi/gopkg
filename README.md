# gopkg

Reusable Go packages. Small, focused, zero-to-minimal third-party dependencies.

## Packages

| Package | Import path | Purpose |
|---|---|---|
| `bus` | `github.com/mahdi-awadi/gopkg/bus` | Pub/sub interface (5 methods) — bring your own adapter |
| `bus/nats` | `github.com/mahdi-awadi/gopkg/bus/nats` | NATS JetStream adapter for `bus.Broker` |
| `communication/provider` | `github.com/mahdi-awadi/gopkg/communication/provider` | Cross-channel notification-delivery contract (+ registry) |
| `communication/sms/twilio` | `github.com/mahdi-awadi/gopkg/communication/sms/twilio` | Twilio SMS adapter |
| `communication/email/sendgrid` | `github.com/mahdi-awadi/gopkg/communication/email/sendgrid` | SendGrid email adapter |
| `communication/email/ses` | `github.com/mahdi-awadi/gopkg/communication/email/ses` | Amazon SES email adapter |
| `communication/push/fcm` | `github.com/mahdi-awadi/gopkg/communication/push/fcm` | Firebase Cloud Messaging push adapter |
| `communication/push/expo` | `github.com/mahdi-awadi/gopkg/communication/push/expo` | Expo push notifications adapter |
| `communication/telegram` | `github.com/mahdi-awadi/gopkg/communication/telegram` | Telegram Bot API adapter |
| `communication/voice/twilio` | `github.com/mahdi-awadi/gopkg/communication/voice/twilio` | Twilio Voice flash-call adapter |
| `communication/whatsapp/meta` | `github.com/mahdi-awadi/gopkg/communication/whatsapp/meta` | Meta Cloud API WhatsApp adapter |
| `communication/whatsapp/twilio` | `github.com/mahdi-awadi/gopkg/communication/whatsapp/twilio` | WhatsApp-over-Twilio adapter |
| `voice/pipeline` | `github.com/mahdi-awadi/gopkg/voice/pipeline` | Realtime telephony-to-LLM voice pipeline |
| `voice/transport/twilio` | `github.com/mahdi-awadi/gopkg/voice/transport/twilio` | Twilio Media Streams transport for `voice/pipeline` |
| `voice/llm/gemini` | `github.com/mahdi-awadi/gopkg/voice/llm/gemini` | Gemini Live LLM adapter for `voice/pipeline` |
| `voice/holdfiller/tone` | `github.com/mahdi-awadi/gopkg/voice/holdfiller/tone` | Mu-law hold-tone filler for `voice/pipeline` |
| `clock` | `github.com/mahdi-awadi/gopkg/clock` | Time abstraction (Real + Mock) for testability |
| `environment` | `github.com/mahdi-awadi/gopkg/environment` | Standard ENVIRONMENT env-var helper |
| `errorsx` | `github.com/mahdi-awadi/gopkg/errorsx` | Error-kind taxonomy with HTTP status mapping |
| `i18n` | `github.com/mahdi-awadi/gopkg/i18n` | `LocalizedString` — JSONB-ready i18n map |
| `id` | `github.com/mahdi-awadi/gopkg/id` | UUIDv7 (time-ordered) generator |
| `identity/jwt` | `github.com/mahdi-awadi/gopkg/identity/jwt` | Generic HMAC-SHA256 JWT sign/verify |
| `observability/health` | `github.com/mahdi-awadi/gopkg/observability/health` | HTTP health-check framework |
| `observability/metrics` | `github.com/mahdi-awadi/gopkg/observability/metrics` | Prometheus metrics exporter |
| `observability/tracing` | `github.com/mahdi-awadi/gopkg/observability/tracing` | OpenTelemetry tracer + OTLP gRPC exporter |
| `money` | `github.com/mahdi-awadi/gopkg/money` | Immutable Money with currency + minor-unit precision |
| `ptr` | `github.com/mahdi-awadi/gopkg/ptr` | Generic `*T` helpers (To, Deref, Or, Equal) |
| `ratelimit` | `github.com/mahdi-awadi/gopkg/ratelimit` | Token-bucket rate limiter |
| `retry` | `github.com/mahdi-awadi/gopkg/retry` | Context-aware exponential backoff with jitter |
| `sqlbuilder` | `github.com/mahdi-awadi/gopkg/sqlbuilder` | Fluent Postgres SELECT builder |
| `storage/r2` | `github.com/mahdi-awadi/gopkg/storage/r2` | Cloudflare R2 (S3-compatible) object storage client |
| `validate` | `github.com/mahdi-awadi/gopkg/validate` | Email / phone / URL / UUID / password validators |
| `cache/lru` | `github.com/mahdi-awadi/gopkg/cache/lru` | Thread-safe generic LRU cache with optional TTL |
| `workerpool` | `github.com/mahdi-awadi/gopkg/workerpool` | Bounded-concurrency goroutine pool |
| `stringcase` | `github.com/mahdi-awadi/gopkg/stringcase` | Case conversions (snake/camel/pascal/kebab/screaming) |
| `crypto/password` | `github.com/mahdi-awadi/gopkg/crypto/password` | bcrypt wrapper for hash/verify/rehash |
| `signals` | `github.com/mahdi-awadi/gopkg/signals` | Graceful-shutdown ctx helpers for SIGINT/SIGTERM |
| `slicex` | `github.com/mahdi-awadi/gopkg/slicex` | Generic Map/Filter/Reduce/Unique/Chunk/GroupBy |
| `httpx/middleware` | `github.com/mahdi-awadi/gopkg/httpx/middleware` | Recover/RequestID/Logger/Timeout/CORS stack |
| `httpx/client` | `github.com/mahdi-awadi/gopkg/httpx/client` | `*http.Client` builder with sane defaults + optional retry |
| `jsonx` | `github.com/mahdi-awadi/gopkg/jsonx` | JSON over HTTP: Decode (size-limited) + Write + Error |
| `mapx` | `github.com/mahdi-awadi/gopkg/mapx` | Generic map helpers (Keys/Merge/Invert/Filter/MapValues) |

## Layout

Multi-module monorepo with Go workspaces (`go.work` at root). Each package has its own `go.mod` and is tagged independently — consumers depend only on what they use.

Tag format: `<package-path>/v<X.Y.Z>` — e.g. `bus/v0.1.0`, `bus/nats/v0.1.0`.

## Versioning

Pre-stable (`v0.x.y`): breaking changes allowed on minor bumps. Pin exactly in `go.mod` until a package hits `v1.0.0`.

After `v1.0.0`: semver strictly enforced. Major bumps live in path (`bus/v2`).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). All commits must pass the CI gate: build, vet, race-test, staticcheck, govulncheck, gofumpt.

## Security

See [SECURITY.md](SECURITY.md) for private vulnerability reporting.

## License

MIT. See [LICENSE](LICENSE).
