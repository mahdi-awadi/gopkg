package slicex

import (
	"reflect"
	"testing"
)

func TestMap(t *testing.T) {
	doubled := Map([]int{1, 2, 3}, func(v int) int { return v * 2 })
	if !reflect.DeepEqual(doubled, []int{2, 4, 6}) {
		t.Fatalf("got %v", doubled)
	}
	if Map([]int(nil), func(v int) int { return v }) != nil {
		t.Fatal("Map(nil) should return nil")
	}
}

func TestFilter(t *testing.T) {
	evens := Filter([]int{1, 2, 3, 4, 5}, func(v int) bool { return v%2 == 0 })
	if !reflect.DeepEqual(evens, []int{2, 4}) {
		t.Fatalf("got %v", evens)
	}
}

func TestReduce(t *testing.T) {
	sum := Reduce([]int{1, 2, 3, 4}, 0, func(acc, v int) int { return acc + v })
	if sum != 10 {
		t.Fatalf("got %d, want 10", sum)
	}
}

func TestUnique(t *testing.T) {
	u := Unique([]int{1, 2, 2, 3, 1, 4})
	if !reflect.DeepEqual(u, []int{1, 2, 3, 4}) {
		t.Fatalf("got %v", u)
	}
	u2 := Unique([]string{"a", "b", "a", "c"})
	if !reflect.DeepEqual(u2, []string{"a", "b", "c"}) {
		t.Fatalf("got %v", u2)
	}
}

func TestChunk(t *testing.T) {
	c := Chunk([]int{1, 2, 3, 4, 5}, 2)
	want := [][]int{{1, 2}, {3, 4}, {5}}
	if !reflect.DeepEqual(c, want) {
		t.Fatalf("got %v", c)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for n=0")
		}
	}()
	Chunk([]int{1}, 0)
}

func TestGroupBy(t *testing.T) {
	type P struct {
		Name string
		Age  int
	}
	p := []P{{"a", 20}, {"b", 30}, {"c", 20}}
	g := GroupBy(p, func(v P) int { return v.Age })
	if len(g[20]) != 2 || len(g[30]) != 1 {
		t.Fatalf("unexpected grouping: %v", g)
	}
}

func TestPartition(t *testing.T) {
	evens, odds := Partition([]int{1, 2, 3, 4}, func(v int) bool { return v%2 == 0 })
	if !reflect.DeepEqual(evens, []int{2, 4}) || !reflect.DeepEqual(odds, []int{1, 3}) {
		t.Fatalf("got %v / %v", evens, odds)
	}
}

func TestAnyAllFind(t *testing.T) {
	s := []int{1, 2, 3}
	if !Any(s, func(v int) bool { return v == 2 }) {
		t.Fatal("Any should find 2")
	}
	if Any(s, func(v int) bool { return v == 99 }) {
		t.Fatal("Any should not find 99")
	}
	if !All(s, func(v int) bool { return v > 0 }) {
		t.Fatal("All positive should be true")
	}
	if All(s, func(v int) bool { return v > 1 }) {
		t.Fatal("All > 1 should be false (1 is included)")
	}

	v, ok := Find(s, func(v int) bool { return v == 2 })
	if !ok || v != 2 {
		t.Fatalf("Find returned %v ok=%v", v, ok)
	}
	_, ok = Find(s, func(v int) bool { return v == 99 })
	if ok {
		t.Fatal("Find should not find 99")
	}
}

func TestAllEmpty(t *testing.T) {
	if !All([]int{}, func(v int) bool { return v > 0 }) {
		t.Fatal("All on empty should be true")
	}
}
