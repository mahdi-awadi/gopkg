# gopkg

Reusable Go packages. Small, focused, zero-to-minimal third-party dependencies.

## Packages

| Package | Import path | Purpose |
|---|---|---|
| `bus` | `github.com/mahdi-awadi/gopkg/bus` | Pub/sub interface (5 methods) — bring your own adapter |
| `bus/nats` | `github.com/mahdi-awadi/gopkg/bus/nats` | NATS JetStream adapter for `bus.Broker` |

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
