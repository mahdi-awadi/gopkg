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
| `clock` | `github.com/mahdi-awadi/gopkg/clock` | Time abstraction (Real + Mock) for testability |
| `environment` | `github.com/mahdi-awadi/gopkg/environment` | Standard ENVIRONMENT env-var helper |
| `errorsx` | `github.com/mahdi-awadi/gopkg/errorsx` | Error-kind taxonomy with HTTP status mapping |
| `i18n` | `github.com/mahdi-awadi/gopkg/i18n` | `LocalizedString` — JSONB-ready i18n map |
| `id` | `github.com/mahdi-awadi/gopkg/id` | UUIDv7 (time-ordered) generator |
| `identity/jwt` | `github.com/mahdi-awadi/gopkg/identity/jwt` | Generic HMAC-SHA256 JWT sign/verify |
| `observability/health` | `github.com/mahdi-awadi/gopkg/observability/health` | HTTP health-check framework |
| `observability/metrics` | `github.com/mahdi-awadi/gopkg/observability/metrics` | Prometheus metrics exporter |
| `observability/tracing` | `github.com/mahdi-awadi/gopkg/observability/tracing` | OpenTelemetry tracer + OTLP gRPC exporter |
| `retry` | `github.com/mahdi-awadi/gopkg/retry` | Context-aware exponential backoff with jitter |
| `storage/r2` | `github.com/mahdi-awadi/gopkg/storage/r2` | Cloudflare R2 (S3-compatible) object storage client |

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
