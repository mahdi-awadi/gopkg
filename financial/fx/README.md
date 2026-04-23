# financial/fx

[![Go Reference](https://pkg.go.dev/badge/github.com/mahdi-awadi/gopkg/financial/fx.svg)](https://pkg.go.dev/github.com/mahdi-awadi/gopkg/financial/fx)

Foreign-exchange interface plus two reference implementations (Memory, Static).

## What it does

Defines the canonical `Rates` interface used across gopkg financial packages. Business code depends on the interface; production wires in a database- or vendor-API-backed adapter; tests use `Static`.

## Install

```bash
go get github.com/mahdi-awadi/gopkg/financial/fx
```

## Quickstart

```go
import "github.com/mahdi-awadi/gopkg/financial/fx"

m := fx.NewMemory()
_ = m.SetPair("USD", "IQD", 1500) // sets both directions
amt, _ := m.Convert(ctx, 10, "USD", "IQD")   // 15000
rate, _ := m.GetRate(ctx, "IQD", "USD")      // 1/1500
```

## API

| Type / Function | Purpose |
|---|---|
| `Rates` | Interface: `GetRate(ctx, from, to)`, `Convert(ctx, amount, from, to)` |
| `Quote` | Rate snapshot with `FetchedAt` for freshness gates |
| `Memory` | Hot-swappable in-process table; safe for concurrent use |
| `Static` | Immutable table; preferred for tests and fixtures |
| `NewMemory()` / `NewStatic(pairs)` | Constructors |
| `(*Memory).Set(from, to, rate)` | One-direction set |
| `(*Memory).SetPair(from, to, rate)` | Both directions: `to→from = 1/rate` |
| `(*Memory).Load(map)` | Atomic bulk replace |
| `(*Memory).Pairs()` | Snapshot (defensive copy) |
| `ErrRateNotFound` | Sentinel for unknown pair |
| `ErrInvalidRate` | Sentinel for non-finite / non-positive input |

## Contract

- Same currency always returns rate `1.0` without a lookup (both impls).
- `GetRate("USD","IQD") = 1500` means **1 USD buys 1500 IQD**.
- Rates must be finite and strictly positive; zero, negative, Inf, and NaN are rejected with `ErrInvalidRate`.
- `Load` is atomic — existing state is preserved if any input value is invalid.

## License

MIT © Mahdi Awadi
