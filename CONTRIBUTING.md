# Contributing

Thanks for the interest. Ground rules:

## Design principles (non-negotiable)

- Accept interfaces, return concrete types
- Small interfaces (1–3 methods preferred)
- `context.Context` first argument on any I/O method
- Functional options for ≥3 optional constructor params
- Typed errors + sentinels (`errors.Is` / `errors.As`)
- Zero package-level mutable state; no `init()` for wiring
- No panics in library code (except genuine programmer errors)
- Docs on every exported symbol (godoc style: first sentence starts with the identifier name)
- Concurrency guarantees documented on every exported type
- Zero mandatory third-party dependencies unless individually justified

## Workflow

1. Open an issue before large PRs.
2. Fork, branch off `main`.
3. Make focused commits. Conventional Commits encouraged (`feat:`, `fix:`, `refactor:`).
4. Run the quality gates locally (Docker commands in `.github/workflows/ci.yml`):
   - `go build ./...`
   - `go vet ./...`
   - `go test -race -coverprofile=cover.out ./...`
   - `golangci-lint run ./...`
   - `govulncheck ./...`
   - `go mod tidy` (must produce no diff)
   - `gofumpt -l .` (must be empty)
5. Open a PR. CI must pass. All exported symbols must have godoc.

## Tagging

Each module is tagged independently with path-scoped tags:

```
bus/v0.1.0
bus/nats/v0.1.0
```

Tags are signed (`git tag -s`).

## Versioning

- `v0.x.y` — pre-stable, breaking changes allowed on minor bumps.
- `v1.0.0` — API freeze. Breaking changes require a new major version in the import path (`bus/v2`).

## License

By contributing you agree the contribution is licensed under the repository's MIT license.
