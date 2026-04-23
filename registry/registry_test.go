package registry

import (
	"errors"
	"sort"
	"sync"
	"testing"
)

func TestRegister_NewKey(t *testing.T) {
	r := New[string, int]()
	if err := r.Register("a", 1); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if r.Len() != 1 {
		t.Errorf("Len=%d, want 1", r.Len())
	}
	v, err := r.Get("a")
	if err != nil || v != 1 {
		t.Errorf("Get=%v,%v want 1,nil", v, err)
	}
}

func TestRegister_DuplicateErrors(t *testing.T) {
	r := New[string, int]()
	_ = r.Register("x", 1)
	err := r.Register("x", 2)
	if !errors.Is(err, ErrAlreadyRegistered) {
		t.Errorf("want ErrAlreadyRegistered, got %v", err)
	}
	// Value should not have been overwritten.
	v, _ := r.Get("x")
	if v != 1 {
		t.Errorf("value mutated on duplicate Register, got %d", v)
	}
}

func TestReplace_OverwritesSilently(t *testing.T) {
	r := New[string, int]()
	_ = r.Register("x", 1)
	prev, replaced := r.Replace("x", 2)
	if !replaced || prev != 1 {
		t.Errorf("Replace=(%d,%v) want (1,true)", prev, replaced)
	}
	v, _ := r.Get("x")
	if v != 2 {
		t.Errorf("after Replace: got %d, want 2", v)
	}
}

func TestReplace_InsertsNew(t *testing.T) {
	r := New[string, int]()
	prev, replaced := r.Replace("new", 5)
	if replaced || prev != 0 {
		t.Errorf("Replace on absent key should return (0,false), got (%d,%v)", prev, replaced)
	}
}

func TestGet_NotFound(t *testing.T) {
	r := New[string, int]()
	_, err := r.Get("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestLookup(t *testing.T) {
	r := New[string, int]()
	_ = r.Register("a", 42)
	v, ok := r.Lookup("a")
	if !ok || v != 42 {
		t.Errorf("Lookup(a)=(%d,%v), want (42,true)", v, ok)
	}
	_, ok = r.Lookup("nope")
	if ok {
		t.Errorf("Lookup on absent should be false")
	}
}

func TestDelete(t *testing.T) {
	r := New[string, int]()
	_ = r.Register("a", 1)
	if !r.Delete("a") {
		t.Errorf("Delete should return true on present key")
	}
	if r.Delete("a") {
		t.Errorf("Delete should return false on absent key")
	}
	if r.Len() != 0 {
		t.Errorf("Len after Delete=%d, want 0", r.Len())
	}
}

func TestKeys(t *testing.T) {
	r := New[string, int]()
	_ = r.Register("a", 1)
	_ = r.Register("b", 2)
	_ = r.Register("c", 3)
	keys := r.Keys()
	sort.Strings(keys)
	want := []string{"a", "b", "c"}
	for i := range want {
		if keys[i] != want[i] {
			t.Errorf("Keys=%v, want %v", keys, want)
			break
		}
	}
}

func TestRange_EarlyExit(t *testing.T) {
	r := New[string, int]()
	for _, k := range []string{"a", "b", "c", "d", "e"} {
		_ = r.Register(k, 1)
	}
	count := 0
	r.Range(func(string, int) bool {
		count++
		return count < 3
	})
	if count != 3 {
		t.Errorf("Range should have visited 3 entries then stopped, got %d", count)
	}
}

func TestRegistry_Concurrent(t *testing.T) {
	r := New[int, int]()
	const N = 100
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func(k int) {
			defer wg.Done()
			_ = r.Register(k, k*2)
		}(i)
	}
	wg.Wait()
	if r.Len() != N {
		t.Errorf("expected %d entries, got %d", N, r.Len())
	}
}

func TestPendingQueue_FlushRunsQueuedFns(t *testing.T) {
	var q PendingQueue[string, int]
	q.Add(func(r *Registry[string, int]) error {
		return r.Register("a", 1)
	})
	q.Add(func(r *Registry[string, int]) error {
		return r.Register("b", 2)
	})

	r := New[string, int]()
	errs := q.Flush(r)
	if len(errs) != 2 || errs[0] != nil || errs[1] != nil {
		t.Errorf("Flush errs=%v want [nil nil]", errs)
	}
	if r.Len() != 2 {
		t.Errorf("registry size after Flush=%d, want 2", r.Len())
	}
}

func TestPendingQueue_AddAfterFlushRunsImmediately(t *testing.T) {
	var q PendingQueue[string, int]
	r := New[string, int]()
	_ = q.Flush(r)

	q.Add(func(r *Registry[string, int]) error {
		return r.Register("late", 99)
	})
	v, _ := r.Get("late")
	if v != 99 {
		t.Errorf("post-flush Add did not run immediately; Get=%d", v)
	}
}

func TestPendingQueue_DoubleFlushNoop(t *testing.T) {
	var q PendingQueue[string, int]
	q.Add(func(r *Registry[string, int]) error { return r.Register("x", 1) })

	r1 := New[string, int]()
	_ = q.Flush(r1)

	r2 := New[string, int]()
	errs := q.Flush(r2)
	if errs != nil {
		t.Errorf("second Flush should return nil slice")
	}
	if r2.Len() != 0 {
		t.Errorf("second target must not receive queued items")
	}
}

func TestPendingQueue_FlushCollectsErrors(t *testing.T) {
	var q PendingQueue[string, int]
	q.Add(func(r *Registry[string, int]) error { return r.Register("x", 1) })
	q.Add(func(r *Registry[string, int]) error { return r.Register("x", 2) }) // dup

	r := New[string, int]()
	errs := q.Flush(r)
	if len(errs) != 2 || errs[0] != nil || errs[1] == nil {
		t.Errorf("want errs=[nil, non-nil], got %v", errs)
	}
}
