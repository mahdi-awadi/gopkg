# registry

[![Go Reference](https://pkg.go.dev/badge/github.com/mahdi-awadi/gopkg/registry.svg)](https://pkg.go.dev/github.com/mahdi-awadi/gopkg/registry)

Concurrent-safe generic registry with pending-registration support.

## What it does

Holds `K → V` pairs behind a `sync.RWMutex`. Supports "pending" registrations: callers (typically `init()` functions in plugin packages) can enqueue registrations before the target registry exists; `Flush` drains the queue when the registry is constructed. This solves the classic plugin/provider wiring problem without requiring a global singleton.

## Install

```bash
go get github.com/mahdi-awadi/gopkg/registry
```

## Quickstart

```go
import "github.com/mahdi-awadi/gopkg/registry"

type Plugin interface{ ... }

// Main wiring
r := registry.New[string, Plugin]()
r.Register("alpha", alphaImpl)
p, err := r.Get("alpha")
```

Self-registering plugins:

```go
// Imported plugin package
var queue registry.PendingQueue[string, Plugin]

func init() {
    queue.Add(func(r *registry.Registry[string, Plugin]) error {
        return r.Register("alpha", &alphaImpl{})
    })
}

// Main package
r := registry.New[string, Plugin]()
queue.Flush(r) // runs every queued registration
```

## API

| Method | Purpose |
|---|---|
| `New[K, V]()` | Construct an empty registry |
| `Register(k, v) error` | Insert; errors if key is taken |
| `Replace(k, v) (prev, replaced)` | Insert or overwrite; silent |
| `Get(k) (v, error)` | Lookup with typed error |
| `Lookup(k) (v, ok)` | Lookup with boolean (no allocation) |
| `Delete(k) bool` | Remove; returns true if removed |
| `Keys()` | Snapshot of keys (order undefined) |
| `Len()` | Entry count |
| `Range(fn)` | Iterate; return false from fn to stop |
| `PendingQueue.Add` | Queue a registration fn |
| `PendingQueue.Flush(r)` | Run queued fns against r; subsequent Adds run immediately |

Typed errors: `ErrAlreadyRegistered`, `ErrNotFound` — match with `errors.Is`.

## Safety

Every method is safe for concurrent use. `Range`'s callback must not call back into the same Registry (deadlock). The zero value of `PendingQueue` is ready to use.

## License

MIT © Mahdi Awadi
