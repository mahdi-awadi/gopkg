package ptr

import "testing"

func TestTo(t *testing.T) {
	s := To("hello")
	if *s != "hello" {
		t.Fatal("To wrong")
	}
	i := To(42)
	if *i != 42 {
		t.Fatal("To[int] wrong")
	}
}

func TestDeref(t *testing.T) {
	if Deref[string](nil) != "" {
		t.Fatal("Deref(nil) should return zero string")
	}
	s := "x"
	if Deref(&s) != "x" {
		t.Fatal("Deref non-nil wrong")
	}
	if Deref[int](nil) != 0 {
		t.Fatal("Deref[int](nil) should return 0")
	}
}

func TestOr(t *testing.T) {
	if Or[string](nil, "fallback") != "fallback" {
		t.Fatal("Or(nil) should return fallback")
	}
	v := "actual"
	if Or(&v, "fallback") != "actual" {
		t.Fatal("Or should return *p when not nil")
	}
}

func TestEqual(t *testing.T) {
	a, b := 5, 5
	c := 6
	if !Equal(&a, &b) {
		t.Fatal("Equal should be true for 5==5")
	}
	if Equal(&a, &c) {
		t.Fatal("Equal should be false for 5!=6")
	}
	if !Equal[int](nil, nil) {
		t.Fatal("Equal(nil, nil) should be true")
	}
	if Equal(&a, nil) {
		t.Fatal("Equal(nonNil, nil) should be false")
	}
}

func TestIsNilOrZero(t *testing.T) {
	if !IsNilOrZero[int](nil) {
		t.Fatal("nil int ptr should be NilOrZero")
	}
	z := 0
	if !IsNilOrZero(&z) {
		t.Fatal("ptr to zero int should be NilOrZero")
	}
	n := 5
	if IsNilOrZero(&n) {
		t.Fatal("ptr to non-zero int should not be NilOrZero")
	}
}
