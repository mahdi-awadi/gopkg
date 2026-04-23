// Package slicex provides generic slice helpers that complement the
// stdlib "slices" package: Map, Filter, Reduce, Unique, Chunk, GroupBy,
// Partition.
//
// Zero third-party deps.
package slicex

// Map applies fn to every element of in and returns the new slice.
func Map[T, R any](in []T, fn func(T) R) []R {
	if len(in) == 0 {
		return nil
	}
	out := make([]R, len(in))
	for i, v := range in {
		out[i] = fn(v)
	}
	return out
}

// Filter returns a new slice containing only elements where keep(v) is true.
func Filter[T any](in []T, keep func(T) bool) []T {
	var out []T
	for _, v := range in {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

// Reduce folds in from the left: fn is called with (accumulator, element).
func Reduce[T, R any](in []T, initial R, fn func(R, T) R) R {
	acc := initial
	for _, v := range in {
		acc = fn(acc, v)
	}
	return acc
}

// Unique returns the input with duplicates removed, preserving order of
// first occurrence.
func Unique[T comparable](in []T) []T {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[T]struct{}, len(in))
	out := make([]T, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

// Chunk splits in into slices of at most size n. The last chunk may be
// shorter. Panics if n <= 0.
func Chunk[T any](in []T, n int) [][]T {
	if n <= 0 {
		panic("slicex: Chunk n must be > 0")
	}
	if len(in) == 0 {
		return nil
	}
	chunks := make([][]T, 0, (len(in)+n-1)/n)
	for i := 0; i < len(in); i += n {
		end := i + n
		if end > len(in) {
			end = len(in)
		}
		chunks = append(chunks, in[i:end])
	}
	return chunks
}

// GroupBy groups elements by the key returned by keyFn.
func GroupBy[T any, K comparable](in []T, keyFn func(T) K) map[K][]T {
	out := make(map[K][]T)
	for _, v := range in {
		k := keyFn(v)
		out[k] = append(out[k], v)
	}
	return out
}

// Partition splits in into two slices: those where keep(v) is true, and
// those where it is false.
func Partition[T any](in []T, keep func(T) bool) (yes, no []T) {
	for _, v := range in {
		if keep(v) {
			yes = append(yes, v)
		} else {
			no = append(no, v)
		}
	}
	return
}

// Any reports whether at least one element satisfies pred.
func Any[T any](in []T, pred func(T) bool) bool {
	for _, v := range in {
		if pred(v) {
			return true
		}
	}
	return false
}

// All reports whether every element satisfies pred. Returns true for empty.
func All[T any](in []T, pred func(T) bool) bool {
	for _, v := range in {
		if !pred(v) {
			return false
		}
	}
	return true
}

// Find returns the first element where pred is true, and whether one was
// found.
func Find[T any](in []T, pred func(T) bool) (T, bool) {
	for _, v := range in {
		if pred(v) {
			return v, true
		}
	}
	var zero T
	return zero, false
}
