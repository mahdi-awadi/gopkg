# pipeline

[![Go Reference](https://pkg.go.dev/badge/github.com/mahdi-awadi/gopkg/pipeline.svg)](https://pkg.go.dev/github.com/mahdi-awadi/gopkg/pipeline)

Generic filter-chain executor for Go — zero third-party dependencies.

## What it does

Runs a list of `Filter[T]` stages over a `[]T`. Each filter can transform, drop, or enrich items. Stages are classified by Criticality (abort-on-failure vs. swallow-on-failure) and by Phase (run-subset support).

## Install

```bash
go get github.com/mahdi-awadi/gopkg/pipeline
```

## Quickstart

```go
import "github.com/mahdi-awadi/gopkg/pipeline"

type myFilter struct{}
func (myFilter) Name() string                      { return "my-filter" }
func (myFilter) Criticality() pipeline.Criticality { return pipeline.Critical }
func (myFilter) Phase() pipeline.Phase             { return 0 }
func (myFilter) Apply(ctx context.Context, items []Offer, meta any) ([]Offer, error) {
    // filter / enrich / transform items
    return items, nil
}

p := pipeline.New[Offer](nil, myFilter{})
out, logs, err := p.Run(ctx, offers, nil)
```

## API

| Type / Function | Purpose |
|---|---|
| `Filter[T]` | Interface every stage implements |
| `Criticality` | `Critical` aborts on error; `Enrichment` swallows errors |
| `Phase` | Opaque tag for `RunPhase` to filter the execution subset |
| `StepLog` | Per-step trace record (duration, before/after count, error) |
| `Logger` | Minimal Debug/Warn/Error contract; zap and slog wrap trivially |
| `NoopLogger{}` | Drop-in default |
| `New[T](logger, filters...)` | Pipeline constructor (defensive-copies filters) |
| `(*Pipeline[T]).Run(ctx, items, meta)` | Run every filter in order |
| `(*Pipeline[T]).RunPhase(ctx, phase, items, meta)` | Run only matching-phase filters |
| `(*Pipeline[T]).Filters()` | Returns a copy of the registered filters |

## Design

- Enrichment failures never bubble up — they're noted in the returned `StepLog` with `Skipped: true`.
- Critical failures return the error plus the logs collected up to and including the failing step.
- `meta` is `any` — callers pass whatever context their filters need (partner ID, trace span, search request, …). The pipeline doesn't interpret it.
- The constructor defensive-copies the filter slice so callers can reuse their input.

## License

MIT © Mahdi Awadi
