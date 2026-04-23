// Package bus defines a small, provider-agnostic pub/sub abstraction.
//
// The bus package contains interfaces and basic configuration types only;
// concrete brokers live in sibling subpackages (e.g.
// github.com/mahdi-awadi/gopkg/bus/nats). A consumer depends on bus for
// its program surface and picks an adapter at wiring time.
//
// # Concurrency
//
// Every Broker implementation returned by an adapter is safe for
// concurrent use by multiple goroutines.
//
// # Cancellation
//
// Every method that performs I/O honors ctx cancellation.
//
// # Zero third-party dependencies (goal)
//
// The v0.1.0 release depends on go.uber.org/zap via the WrapZap helper.
// A v0.2.0 will split WrapZap into a separate bus/zaplog subpackage so
// the core bus module can be stdlib-only. Consumers who don't need the
// zap adapter today can import bus and use NoopLogger.
package bus
