package i18n

import "testing"

func TestLocalizedString_Get(t *testing.T) {
	tests := []struct {
		name string
		ls   LocalizedString
		lang string
		want string
	}{
		{"exact match", LocalizedString{"en": "Hello", "ar": "مرحبا"}, "ar", "مرحبا"},
		{"fallback to en", LocalizedString{"en": "Hello", "ar": "مرحبا"}, "ku", "Hello"},
		{"fallback to first when en missing", LocalizedString{"ar": "مرحبا"}, "ku", "مرحبا"},
		{"empty map", LocalizedString{}, "en", ""},
		{"nil map", nil, "en", ""},
		{"empty string in target lang falls through", LocalizedString{"ku": "", "en": "Hello"}, "ku", "Hello"},
		{"all empty strings returns empty", LocalizedString{"en": "", "ar": ""}, "en", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ls.Get(tt.lang); got != tt.want {
				t.Errorf("Get(%q) = %q, want %q", tt.lang, got, tt.want)
			}
		})
	}
}

func TestLocalizedString_ScanValue(t *testing.T) {
	original := LocalizedString{"en": "Hello", "ar": "مرحبا"}
	v, err := original.Value()
	if err != nil {
		t.Fatalf("Value() error: %v", err)
	}
	bytes, ok := v.([]byte)
	if !ok {
		t.Fatalf("Value() returned %T, want []byte", v)
	}

	var roundTripped LocalizedString
	if err := roundTripped.Scan(bytes); err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if roundTripped["en"] != "Hello" || roundTripped["ar"] != "مرحبا" {
		t.Errorf("round-trip mismatch: %v", roundTripped)
	}
}

func TestLocalizedString_ScanNil(t *testing.T) {
	var ls LocalizedString
	if err := ls.Scan(nil); err != nil {
		t.Fatalf("Scan(nil) error: %v", err)
	}
	if ls == nil || len(ls) != 0 {
		t.Errorf("Scan(nil) should produce empty map, got %v", ls)
	}
}

func TestLocalizedString_ScanString(t *testing.T) {
	// Some PostgreSQL drivers hand back JSONB as a string rather than []byte.
	// The implementation must handle both.
	var ls LocalizedString
	if err := ls.Scan(`{"en":"Hi","ar":"مرحبا"}`); err != nil {
		t.Fatalf("Scan(string) error: %v", err)
	}
	if ls["en"] != "Hi" || ls["ar"] != "مرحبا" {
		t.Errorf("Scan(string) round-trip mismatch: %v", ls)
	}
}

func TestLocalizedString_ScanUnsupportedType(t *testing.T) {
	var ls LocalizedString
	err := ls.Scan(42)
	if err == nil {
		t.Fatal("Scan(int) should error, got nil")
	}
}

func TestLocalizedString_ValueNilMap(t *testing.T) {
	// A nil LocalizedString must produce a valid empty JSON object so it can
	// be written to a NOT NULL JSONB column without bypassing the constraint.
	var ls LocalizedString
	v, err := ls.Value()
	if err != nil {
		t.Fatalf("Value() on nil error: %v", err)
	}
	// Implementation may return either a string "{}" or []byte("{}").
	// Both serialize to the same JSON; assert on string form to be tolerant.
	switch got := v.(type) {
	case string:
		if got != "{}" {
			t.Errorf("nil Value() = %q, want \"{}\"", got)
		}
	case []byte:
		if string(got) != "{}" {
			t.Errorf("nil Value() = %q, want \"{}\"", string(got))
		}
	default:
		t.Errorf("nil Value() returned unexpected type %T", v)
	}
}

func TestLocalizedString_GetEmptyLangArg(t *testing.T) {
	// An empty language argument should fall through to the en fallback,
	// not be treated as a literal "" key lookup-with-success.
	ls := LocalizedString{"en": "Hello", "ar": "مرحبا"}
	if got := ls.Get(""); got != "Hello" {
		t.Errorf("Get(\"\") = %q, want \"Hello\"", got)
	}
}
