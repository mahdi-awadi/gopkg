package clock

import (
	"testing"
	"time"
)

func TestReal_Now(t *testing.T) {
	r := Real{}
	before := time.Now()
	got := r.Now()
	after := time.Now()
	if got.Before(before) || got.After(after) {
		t.Fatalf("Real.Now out of range: %v not in [%v,%v]", got, before, after)
	}
}

func TestMock_AdvanceMovesNow(t *testing.T) {
	start := time.Date(2026, 4, 23, 0, 0, 0, 0, time.UTC)
	m := NewMock(start)
	m.Advance(90 * time.Second)
	if !m.Now().Equal(start.Add(90 * time.Second)) {
		t.Fatalf("got %v, want %v", m.Now(), start.Add(90*time.Second))
	}
}

func TestMock_AfterFiresOnAdvance(t *testing.T) {
	m := NewMock(time.Unix(0, 0).UTC())
	ch := m.After(5 * time.Second)

	select {
	case <-ch:
		t.Fatal("After fired before Advance")
	default:
	}

	m.Advance(3 * time.Second)
	select {
	case <-ch:
		t.Fatal("After fired early")
	default:
	}

	m.Advance(2 * time.Second)
	select {
	case <-ch:
		// expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("After did not fire after Advance")
	}
}

func TestMock_AfterZeroFiresImmediately(t *testing.T) {
	m := NewMock(time.Unix(0, 0).UTC())
	ch := m.After(0)
	select {
	case <-ch:
		// expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("After(0) did not fire")
	}
}

func TestMock_SetFiresDueWaiters(t *testing.T) {
	m := NewMock(time.Unix(0, 0).UTC())
	ch := m.After(10 * time.Second)
	m.Set(time.Unix(11, 0).UTC())
	select {
	case <-ch:
		// expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Set did not fire due waiter")
	}
}

func TestMock_Since(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	m := NewMock(start)
	m.Advance(90 * time.Second)
	if m.Since(start) != 90*time.Second {
		t.Fatalf("got %v, want 90s", m.Since(start))
	}
}

func TestClockInterface_Real(t *testing.T) {
	var c Clock = Real{}
	_ = c.Now()
	_ = c.Since(time.Now())
	_ = c.After(0)
}

func TestClockInterface_Mock(t *testing.T) {
	var c Clock = NewMock(time.Now())
	_ = c.Now()
	_ = c.Since(time.Now())
	_ = c.After(0)
}
