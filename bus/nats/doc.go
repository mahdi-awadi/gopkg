// Package nats implements bus.Broker on top of NATS JetStream.
//
// Construct a Broker with NewBroker; the returned value is safe for
// concurrent use.
//
// # Server compatibility
//
// Requires a NATS server with JetStream enabled, ≥ 2.10.
//
// # Cancellation and closing
//
// Every I/O method honors ctx cancellation. Close is idempotent.
package nats
