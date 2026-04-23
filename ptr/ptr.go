// Package ptr provides tiny generic helpers for working with pointers to
// values. Useful primarily for JSON/SQL struct fields where `*T` is the
// canonical way to model nullability.
//
// Zero third-party deps.
package ptr

// To returns a pointer to v. Useful for building structs with pointer
// fields inline: &Struct{Name: ptr.To("alice")}.
func To[T any](v T) *T {
	return &v
}

// Deref returns the value pointed to by p, or the zero value of T if p is nil.
func Deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// Or returns the value pointed to by p, or `fallback` if p is nil.
func Or[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}

// Equal reports whether two pointers point to equal values. Two nil
// pointers are equal; a nil and a non-nil are not.
func Equal[T comparable](a, b *T) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	}
	return *a == *b
}

// IsNilOrZero reports whether p is nil, or points to the zero value of T.
func IsNilOrZero[T comparable](p *T) bool {
	if p == nil {
		return true
	}
	var zero T
	return *p == zero
}
