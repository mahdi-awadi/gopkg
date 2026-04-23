package fx

import (
	"context"
	"errors"
	"math"
	"sync"
	"testing"
)

func TestMemory_SameCurrencyReturnsOne(t *testing.T) {
	m := NewMemory()
	r, err := m.GetRate(context.Background(), "USD", "USD")
	if err != nil || r != 1.0 {
		t.Errorf("USD→USD got (%v, %v), want (1.0, nil)", r, err)
	}
}

func TestMemory_RateNotFound(t *testing.T) {
	m := NewMemory()
	_, err := m.GetRate(context.Background(), "USD", "IQD")
	if !errors.Is(err, ErrRateNotFound) {
		t.Errorf("want ErrRateNotFound, got %v", err)
	}
}

func TestMemory_SetAndGet(t *testing.T) {
	m := NewMemory()
	if err := m.Set("USD", "IQD", 1500); err != nil {
		t.Fatalf("Set: %v", err)
	}
	r, err := m.GetRate(context.Background(), "USD", "IQD")
	if err != nil || r != 1500 {
		t.Errorf("got (%v, %v), want (1500, nil)", r, err)
	}
	// Reverse direction not set.
	_, err = m.GetRate(context.Background(), "IQD", "USD")
	if !errors.Is(err, ErrRateNotFound) {
		t.Errorf("reverse should be absent, got %v", err)
	}
}

func TestMemory_SetPairBidirectional(t *testing.T) {
	m := NewMemory()
	if err := m.SetPair("USD", "IQD", 1500); err != nil {
		t.Fatalf("SetPair: %v", err)
	}
	forward, _ := m.GetRate(context.Background(), "USD", "IQD")
	if forward != 1500 {
		t.Errorf("forward rate=%v, want 1500", forward)
	}
	backward, _ := m.GetRate(context.Background(), "IQD", "USD")
	if math.Abs(backward-(1.0/1500)) > 1e-12 {
		t.Errorf("backward rate=%v, want 1/1500", backward)
	}
}

func TestMemory_Convert(t *testing.T) {
	m := NewMemory()
	_ = m.Set("USD", "IQD", 1500)
	got, err := m.Convert(context.Background(), 10, "USD", "IQD")
	if err != nil || got != 15000 {
		t.Errorf("Convert=(%v,%v), want (15000, nil)", got, err)
	}
}

func TestMemory_InvalidRateRejected(t *testing.T) {
	m := NewMemory()
	for _, bad := range []float64{0, -1, math.Inf(1), math.NaN()} {
		if err := m.Set("A", "B", bad); !errors.Is(err, ErrInvalidRate) {
			t.Errorf("Set(%v) should be rejected, got %v", bad, err)
		}
		if err := m.SetPair("A", "B", bad); !errors.Is(err, ErrInvalidRate) {
			t.Errorf("SetPair(%v) should be rejected, got %v", bad, err)
		}
	}
}

func TestMemory_Load(t *testing.T) {
	m := NewMemory()
	_ = m.Set("keep", "me", 1)
	err := m.Load(map[string]float64{
		"USD>IQD": 1500,
		"EUR>USD": 1.1,
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Previous entry should be wiped.
	if _, err := m.GetRate(context.Background(), "keep", "me"); !errors.Is(err, ErrRateNotFound) {
		t.Errorf("Load must replace the whole table")
	}
	if r, _ := m.GetRate(context.Background(), "USD", "IQD"); r != 1500 {
		t.Errorf("USD>IQD=%v, want 1500", r)
	}
}

func TestMemory_LoadPreservesOnInvalid(t *testing.T) {
	m := NewMemory()
	_ = m.Set("USD", "IQD", 1500)
	err := m.Load(map[string]float64{
		"A>B": 2,
		"C>D": math.NaN(), // invalid
	})
	if !errors.Is(err, ErrInvalidRate) {
		t.Errorf("want ErrInvalidRate, got %v", err)
	}
	// Old state should be intact.
	if r, _ := m.GetRate(context.Background(), "USD", "IQD"); r != 1500 {
		t.Errorf("Load must preserve state on error; got %v", r)
	}
}

func TestMemory_ContextCancellation(t *testing.T) {
	m := NewMemory()
	_ = m.Set("A", "B", 2)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := m.GetRate(ctx, "A", "B")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled ctx should fail, got %v", err)
	}
}

func TestMemory_Pairs(t *testing.T) {
	m := NewMemory()
	_ = m.Set("A", "B", 2)
	_ = m.Set("B", "C", 3)
	pairs := m.Pairs()
	if len(pairs) != 2 {
		t.Errorf("len(pairs)=%d, want 2", len(pairs))
	}
	// Mutating the returned map must not affect the store.
	pairs["A>B"] = 99
	if r, _ := m.GetRate(context.Background(), "A", "B"); r != 2 {
		t.Errorf("Pairs must return a defensive copy")
	}
}

func TestMemory_Concurrent(t *testing.T) {
	m := NewMemory()
	_ = m.Set("A", "B", 2)
	var wg sync.WaitGroup
	const N = 50
	wg.Add(N * 2)
	for i := 0; i < N; i++ {
		go func() { defer wg.Done(); _, _ = m.GetRate(context.Background(), "A", "B") }()
		go func(i int) {
			defer wg.Done()
			_ = m.Set("A", "B", float64(i)+1)
		}(i)
	}
	wg.Wait()
}

func TestStatic_Immutable(t *testing.T) {
	src := map[string]float64{"A>B": 2}
	s, err := NewStatic(src)
	if err != nil {
		t.Fatalf("NewStatic: %v", err)
	}
	src["A>B"] = 99 // must not leak into Static
	r, _ := s.GetRate(context.Background(), "A", "B")
	if r != 2 {
		t.Errorf("Static must defensive-copy input; got %v", r)
	}
}

func TestStatic_Convert(t *testing.T) {
	s, _ := NewStatic(map[string]float64{"A>B": 0.5})
	got, _ := s.Convert(context.Background(), 100, "A", "B")
	if got != 50 {
		t.Errorf("Convert=%v, want 50", got)
	}
}

func TestStatic_InvalidRateRejected(t *testing.T) {
	_, err := NewStatic(map[string]float64{"A>B": 0})
	if !errors.Is(err, ErrInvalidRate) {
		t.Errorf("want ErrInvalidRate, got %v", err)
	}
}

// Compile-time assertions that both implementations satisfy the interface.
var (
	_ Rates = (*Memory)(nil)
	_ Rates = (*Static)(nil)
)
