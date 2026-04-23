// Package mapx provides generic map helpers that complement the stdlib
// "maps" package: Keys (sorted + unsorted), Values, Merge, Invert,
// Filter, MapValues, Equal (deep).
//
// Zero third-party deps.
package mapx

import "sort"

// Keys returns the keys of m in no particular order.
func Keys[K comparable, V any](m map[K]V) []K {
	if len(m) == 0 {
		return nil
	}
	out := make([]K, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// Ordered constraint lets us sort.
type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 | ~string
}

// KeysSorted returns the keys of m sorted ascending.
func KeysSorted[K Ordered, V any](m map[K]V) []K {
	ks := Keys(m)
	sort.Slice(ks, func(i, j int) bool { return ks[i] < ks[j] })
	return ks
}

// Values returns the values of m in no particular order.
func Values[K comparable, V any](m map[K]V) []V {
	if len(m) == 0 {
		return nil
	}
	out := make([]V, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

// Merge returns a new map containing all entries from all inputs.
// Later entries override earlier ones.
func Merge[K comparable, V any](maps ...map[K]V) map[K]V {
	size := 0
	for _, m := range maps {
		size += len(m)
	}
	out := make(map[K]V, size)
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// Invert swaps keys and values. If input has duplicate values, the
// surviving value's key is not deterministic.
func Invert[K, V comparable](m map[K]V) map[V]K {
	out := make(map[V]K, len(m))
	for k, v := range m {
		out[v] = k
	}
	return out
}

// Filter returns a new map containing only entries where keep(k, v) is true.
func Filter[K comparable, V any](m map[K]V, keep func(K, V) bool) map[K]V {
	out := make(map[K]V)
	for k, v := range m {
		if keep(k, v) {
			out[k] = v
		}
	}
	return out
}

// MapValues returns a new map with values transformed by fn.
func MapValues[K comparable, V, R any](m map[K]V, fn func(K, V) R) map[K]R {
	out := make(map[K]R, len(m))
	for k, v := range m {
		out[k] = fn(k, v)
	}
	return out
}

// Equal reports whether two maps have the same keys and equal values.
// Both nil and empty maps are equal.
func Equal[K, V comparable](a, b map[K]V) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok || bv != av {
			return false
		}
	}
	return true
}
