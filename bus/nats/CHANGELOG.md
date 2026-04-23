# Changelog

All notable changes to `github.com/mahdi-awadi/gopkg/bus/nats` are documented here.

## [Unreleased]

## [0.1.0] - 2026-04-23

### Added
- `NewBroker(cfg *bus.Config, logger bus.Logger) (bus.Broker, error)` constructor
- `Broker` type implementing `bus.Broker` on top of NATS JetStream
  - `PublishRaw` via `jetstream.Publish`
  - `Subscribe` with durable consumer creation (Explicit Ack, 30s AckWait, 5 retries, DeliverNew)
  - `Health` via `*nats.Conn.IsConnected()`
  - `Drain` via `*nats.Conn.Drain()`
  - `Close` (idempotent)
- JSON serialization of `PublishRaw` payload
- Compile-time check that `*Broker` satisfies `bus.Broker`

### Requirements
- Go 1.23+
- NATS server ≥ 2.10 with JetStream enabled
- Streams/consumers provisioned externally
