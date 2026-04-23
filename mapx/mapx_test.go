package mapx

import (
	"reflect"
	"testing"
)

func TestKeys_EmptyAndPresent(t *testing.T) {
	if Keys[string, int](nil) != nil {
		t.Fatal("Keys(nil) should be nil")
	}
	m := map[string]int{"a": 1, "b": 2}
	ks := Keys(m)
	if len(ks) != 2 {
		t.Fatalf("len=%d", len(ks))
	}
}

func TestKeysSorted_IsAscending(t *testing.T) {
	ks := KeysSorted(map[string]int{"c": 3, "a": 1, "b": 2})
	if !reflect.DeepEqual(ks, []string{"a", "b", "c"}) {
		t.Fatalf("got %v", ks)
	}
	is := KeysSorted(map[int]string{3: "c", 1: "a", 2: "b"})
	if !reflect.DeepEqual(is, []int{1, 2, 3}) {
		t.Fatalf("got %v", is)
	}
}

func TestValues_LengthMatches(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	vs := Values(m)
	if len(vs) != 3 {
		t.Fatalf("len=%d", len(vs))
	}
}

func TestMerge_LaterOverrides(t *testing.T) {
	a := map[string]int{"a": 1, "b": 2}
	b := map[string]int{"b": 20, "c": 3}
	c := Merge(a, b)
	want := map[string]int{"a": 1, "b": 20, "c": 3}
	if !Equal(c, want) {
		t.Fatalf("got %v, want %v", c, want)
	}
}

func TestInvert(t *testing.T) {
	inv := Invert(map[string]int{"a": 1, "b": 2})
	if inv[1] != "a" || inv[2] != "b" {
		t.Fatalf("got %v", inv)
	}
}

func TestFilter(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	even := Filter(m, func(_ string, v int) bool { return v%2 == 0 })
	if !Equal(even, map[string]int{"b": 2}) {
		t.Fatalf("got %v", even)
	}
}

func TestMapValues(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	d := MapValues(m, func(_ string, v int) int { return v * 10 })
	if !Equal(d, map[string]int{"a": 10, "b": 20}) {
		t.Fatalf("got %v", d)
	}
}

func TestEqual_NilAndEmpty(t *testing.T) {
	if !Equal[string, int](nil, nil) {
		t.Fatal("nil == nil")
	}
	if !Equal(map[string]int{}, nil) {
		t.Fatal("empty == nil")
	}
	if Equal(map[string]int{"a": 1}, map[string]int{"a": 2}) {
		t.Fatal("different values")
	}
	if Equal(map[string]int{"a": 1}, map[string]int{"b": 1}) {
		t.Fatal("different keys")
	}
}
