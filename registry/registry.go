// Package registry provides a concurrent-safe generic key/value registry
// with support for "pending" registrations — entries that are staged
// during package init() and committed later, once wiring code has
// constructed the registry.
//
// Typical use: a provider/plugin/adapter architecture where each
// implementation wants to self-register in its own init() without
// forcing the main package to import every implementation explicitly
// (blank imports suffice). Pending registrations sidestep the classic
// "init() runs before the registry exists" ordering problem.
//
// The package has zero third-party dependencies.
package registry

import (
	"errors"
	"fmt"
	"sync"
)

// ErrAlreadyRegistered is returned by Register when the key is taken.
var ErrAlreadyRegistered = errors.New("registry: key already registered")

// ErrNotFound is returned by Get when the key is unknown.
var ErrNotFound = errors.New("registry: key not found")

// Registry is a concurrent-safe map[K]V with Register / Get / Keys / Len.
// Registry is NOT copyable after first use; pass it by pointer.
type Registry[K comparable, V any] struct {
	mu      sync.RWMutex
	entries map[K]V
}

// New constructs an empty Registry.
func New[K comparable, V any]() *Registry[K, V] {
	return &Registry[K, V]{entries: make(map[K]V)}
}

// Register inserts value v under key k. Returns ErrAlreadyRegistered if
// the key is already present (an overwrite is treated as a caller bug).
// Use Replace if overwrite is intentional.
func (r *Registry[K, V]) Register(k K, v V) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.entries[k]; exists {
		return fmt.Errorf("%w: %v", ErrAlreadyRegistered, k)
	}
	r.entries[k] = v
	return nil
}

// Replace inserts or overwrites the value at key k. Always succeeds.
// Returns the previous value (zero value if the key was absent) and a
// boolean indicating whether a replacement occurred.
func (r *Registry[K, V]) Replace(k K, v V) (prev V, replaced bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.entries[k]; ok {
		prev = existing
		replaced = true
	}
	r.entries[k] = v
	return prev, replaced
}

// Get returns the value at key k. Returns ErrNotFound if absent.
func (r *Registry[K, V]) Get(k K) (V, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.entries[k]
	if !ok {
		var zero V
		return zero, fmt.Errorf("%w: %v", ErrNotFound, k)
	}
	return v, nil
}

// Lookup is a panic-free zero-value variant: returns (v, true) if present,
// (zero, false) otherwise. Preferred over Get when absence is expected.
func (r *Registry[K, V]) Lookup(k K) (V, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.entries[k]
	return v, ok
}

// Delete removes the entry at key k. Returns true if an entry was removed.
func (r *Registry[K, V]) Delete(k K) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.entries[k]; !ok {
		return false
	}
	delete(r.entries, k)
	return true
}

// Keys returns a snapshot of all registered keys. Order is undefined.
func (r *Registry[K, V]) Keys() []K {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]K, 0, len(r.entries))
	for k := range r.entries {
		out = append(out, k)
	}
	return out
}

// Len returns the number of entries. Safe for concurrent use.
func (r *Registry[K, V]) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.entries)
}

// Range invokes fn(k, v) for every entry. Iteration is safe for
// concurrent reads but fn must not call back into the same Registry
// (deadlock). Iteration order is undefined. Return false from fn to
// stop early.
func (r *Registry[K, V]) Range(fn func(k K, v V) bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for k, v := range r.entries {
		if !fn(k, v) {
			return
		}
	}
}

// PendingQueue buffers registration functions that must run after the
// target Registry is constructed. Useful for init()-time self-
// registration when the target registry can only exist later.
//
// The zero value is ready to use. Once Flush is called, new Add calls
// execute the supplied function immediately on the target registry.
type PendingQueue[K comparable, V any] struct {
	mu      sync.Mutex
	target  *Registry[K, V]
	pending []func(r *Registry[K, V]) error
}

// Add queues fn for execution when Flush is eventually called with a
// target registry. If Flush has already run, fn executes immediately
// against the flushed target.
//
// Any error returned by fn is ignored by the queue; callers that care
// about error propagation should use Flush's return slice (which holds
// the errors from the initial flush) and arrange their own error
// surface for post-flush Adds. This matches the common "self-register
// in init()" pattern where panicking or logging is the caller's call.
func (q *PendingQueue[K, V]) Add(fn func(r *Registry[K, V]) error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.target != nil {
		_ = fn(q.target)
		return
	}
	q.pending = append(q.pending, fn)
}

// Flush sets the target registry and runs every queued function
// against it. Returns the slice of errors (aligned with the pending
// slice; entries are nil for successful registrations).
//
// Subsequent Adds bypass the queue and run immediately against r.
// Calling Flush twice on the same queue is a caller error and returns
// a nil slice the second time (all subsequent work goes through Add).
func (q *PendingQueue[K, V]) Flush(r *Registry[K, V]) []error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.target != nil {
		return nil
	}
	q.target = r
	errs := make([]error, 0, len(q.pending))
	for _, fn := range q.pending {
		errs = append(errs, fn(r))
	}
	q.pending = nil
	return errs
}
