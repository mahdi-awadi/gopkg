package lru

import (
	"testing"
	"time"
)

func TestSetAndGet(t *testing.T) {
	c := New[string, int](3, 0)
	c.Set("a", 1)
	c.Set("b", 2)

	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Fatalf("a=%v ok=%v", v, ok)
	}
	if v, ok := c.Get("missing"); ok || v != 0 {
		t.Fatalf("missing should return zero,false; got %v,%v", v, ok)
	}
}

func TestEvictsLRU(t *testing.T) {
	c := New[string, int](2, 0)
	c.Set("a", 1)
	c.Set("b", 2)
	// access "a" → now "b" is LRU
	_, _ = c.Get("a")
	c.Set("c", 3) // should evict "b"

	if _, ok := c.Get("b"); ok {
		t.Fatal("b should have been evicted")
	}
	if _, ok := c.Get("a"); !ok {
		t.Fatal("a should still be present")
	}
}

func TestTTL_Expires(t *testing.T) {
	c := New[string, int](10, 100*time.Millisecond)
	start := time.Now()
	c.now = func() time.Time { return start }
	c.Set("a", 1)

	// Within TTL
	c.now = func() time.Time { return start.Add(50 * time.Millisecond) }
	if _, ok := c.Get("a"); !ok {
		t.Fatal("should be present within TTL")
	}

	// After TTL
	c.now = func() time.Time { return start.Add(150 * time.Millisecond) }
	if _, ok := c.Get("a"); ok {
		t.Fatal("should have expired")
	}
}

func TestSet_Replaces(t *testing.T) {
	c := New[string, int](3, 0)
	c.Set("a", 1)
	c.Set("a", 2)
	if v, ok := c.Get("a"); !ok || v != 2 {
		t.Fatalf("expected 2, got %v", v)
	}
	if c.Len() != 1 {
		t.Fatalf("Len=%d, want 1", c.Len())
	}
}

func TestDelete(t *testing.T) {
	c := New[string, int](3, 0)
	c.Set("a", 1)
	c.Delete("a")
	if _, ok := c.Get("a"); ok {
		t.Fatal("a should be gone")
	}
}

func TestClear(t *testing.T) {
	c := New[string, int](3, 0)
	c.Set("a", 1)
	c.Set("b", 2)
	c.Clear()
	if c.Len() != 0 {
		t.Fatalf("Len=%d after Clear", c.Len())
	}
}

func TestCapacityMinimum(t *testing.T) {
	c := New[string, int](0, 0) // should clamp to 1
	c.Set("a", 1)
	c.Set("b", 2)
	if c.Len() != 1 {
		t.Fatalf("Len=%d, want 1", c.Len())
	}
}
