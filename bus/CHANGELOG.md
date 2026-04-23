# Changelog

All notable changes to `github.com/mahdi-awadi/gopkg/bus` are documented here. Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versioning: [Semver](https://semver.org/).

## [Unreleased]

## [0.1.0] - 2026-04-23

### Added
- `Broker` interface with 5 methods: `PublishRaw`, `Subscribe`, `Health`, `Drain`, `Close`
- `Message`, `MessageHandler`, `Subscription` types
- `Logger` interface (`Info`, `Error`) + `NoopLogger` zero-value + `WrapZap(*zap.Logger)` helper
- `Config` struct with `URL`, `User`, `Password`, `ServiceName`, `Type` fields
- `PublishOption` + `WithOrderingKey`, `WithHeaders`
- Companion `bus/nats` package providing the NATS JetStream adapter

### Known deviations from design target
- `WrapZap` lives in this module, pulling `go.uber.org/zap` as a required dependency. A future v0.2.0 will move `WrapZap` into a sibling `bus/zaplog` module so the core `bus` module becomes stdlib-only.
