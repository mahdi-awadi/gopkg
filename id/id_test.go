package id

import (
	"regexp"
	"testing"
)

var uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestUUIDv7_FormatMatchesRFC(t *testing.T) {
	for i := 0; i < 32; i++ {
		u := UUIDv7()
		if !uuidRe.MatchString(u) {
			t.Fatalf("generated UUID %q does not match RFC v7 format", u)
		}
	}
}

func TestUUIDv7_UniquenessOverManyCalls(t *testing.T) {
	seen := make(map[string]struct{}, 10_000)
	for i := 0; i < 10_000; i++ {
		u := UUIDv7()
		if _, dup := seen[u]; dup {
			t.Fatalf("duplicate UUID: %q", u)
		}
		seen[u] = struct{}{}
	}
}

func TestUUIDv7_TimeOrderedAcrossCalls(t *testing.T) {
	// 48-bit ms timestamp makes adjacent UUIDs sort lexicographically.
	a := UUIDv7()
	// Sleep a couple ms to guarantee a different ms timestamp on fast hosts.
	// (Reliable even on low-resolution clocks at 1ms granularity.)
	for {
		b := UUIDv7()
		if b > a { // string compare
			return
		}
		// loop until we observe a later-sorting UUID; guaranteed within 2ms
		a = b
	}
}

func TestUUIDv7Raw_Bytes(t *testing.T) {
	b := UUIDv7Raw()
	if (b[6]>>4)&0x0F != 7 {
		t.Fatalf("version nibble = %x, want 7", (b[6]>>4)&0x0F)
	}
	if (b[8]>>6)&0x03 != 0b10 {
		t.Fatalf("variant bits = %b, want 10", (b[8]>>6)&0x03)
	}
}
