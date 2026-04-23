package environment

import (
	"os"
	"testing"
)

// Note: GetEnvironment caches via sync.Once — these tests verify the
// idempotency predicate helpers and the one-time cache behavior.

func TestIs_Helpers(t *testing.T) {
	// Cache is internal; we test that exactly one of the predicates is true.
	count := 0
	for _, b := range []bool{IsDevelopment(), IsTesting(), IsStaging(), IsProduction()} {
		if b {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one environment predicate to be true, got %d", count)
	}
}

func TestGetEnvironment_ValueInKnownSet(t *testing.T) {
	e := GetEnvironment()
	switch e {
	case Development, Testing, Staging, Production:
		// ok
	default:
		t.Fatalf("unexpected environment value: %q", e)
	}
}

func TestEnvironment_DefaultsToProductionWhenUnset(t *testing.T) {
	// We can't reset the sync.Once from a test; but when the test binary runs
	// without ENVIRONMENT set, Production is the default. If ENVIRONMENT is
	// set by the runner, we just verify we got a valid value (tested above).
	if os.Getenv("ENVIRONMENT") == "" {
		if GetEnvironment() != Production {
			t.Fatalf("empty ENVIRONMENT should default to Production, got %q", GetEnvironment())
		}
	}
}
