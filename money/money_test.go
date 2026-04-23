package money

import (
	"errors"
	"testing"
)

func TestNew_UnknownCurrency(t *testing.T) {
	_, err := New(10.0, "XYZ")
	if !errors.Is(err, ErrUnknownCurrency) {
		t.Fatalf("expected ErrUnknownCurrency, got %v", err)
	}
}

func TestNew_USDAndKWDScale(t *testing.T) {
	usd, err := New(1.50, "USD")
	if err != nil {
		t.Fatal(err)
	}
	if usd.Minor() != 150 {
		t.Fatalf("USD 1.50 → minor=%d, want 150", usd.Minor())
	}

	kwd, _ := New(0.500, "KWD")
	if kwd.Minor() != 500 {
		t.Fatalf("KWD 0.500 → minor=%d, want 500", kwd.Minor())
	}

	iqd, _ := New(5000, "IQD")
	if iqd.Minor() != 5000 {
		t.Fatalf("IQD 5000 → minor=%d, want 5000 (scale 0)", iqd.Minor())
	}
}

func TestFromMinor(t *testing.T) {
	m, _ := FromMinor(150, "USD")
	if m.Minor() != 150 {
		t.Fatal("wrong minor")
	}
}

func TestAdd_SameCurrency(t *testing.T) {
	a, _ := FromMinor(100, "USD")
	b, _ := FromMinor(25, "USD")
	sum, err := a.Add(b)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Minor() != 125 {
		t.Fatalf("got %d, want 125", sum.Minor())
	}
}

func TestAdd_CurrencyMismatch(t *testing.T) {
	a, _ := FromMinor(100, "USD")
	b, _ := FromMinor(100, "EUR")
	_, err := a.Add(b)
	if !errors.Is(err, ErrCurrencyMismatch) {
		t.Fatalf("expected ErrCurrencyMismatch, got %v", err)
	}
}

func TestSub(t *testing.T) {
	a, _ := FromMinor(500, "USD")
	b, _ := FromMinor(125, "USD")
	diff, _ := a.Sub(b)
	if diff.Minor() != 375 {
		t.Fatalf("got %d, want 375", diff.Minor())
	}
}

func TestMulInt(t *testing.T) {
	a, _ := FromMinor(50, "USD")
	total, _ := a.MulInt(4)
	if total.Minor() != 200 {
		t.Fatalf("got %d, want 200", total.Minor())
	}
}

func TestDivInt_DivideByZero(t *testing.T) {
	a, _ := FromMinor(100, "USD")
	_, err := a.DivInt(0)
	if !errors.Is(err, ErrDivideByZero) {
		t.Fatalf("expected ErrDivideByZero, got %v", err)
	}
}

func TestEqualAndCmp(t *testing.T) {
	a, _ := FromMinor(100, "USD")
	b, _ := FromMinor(100, "USD")
	c, _ := FromMinor(200, "USD")

	if !a.Equal(b) {
		t.Fatal("100 USD == 100 USD")
	}
	ord, _ := a.Cmp(c)
	if ord != -1 {
		t.Fatalf("100 < 200, got %d", ord)
	}
}

func TestNegate(t *testing.T) {
	a, _ := FromMinor(250, "USD")
	n := a.Negate()
	if n.Minor() != -250 {
		t.Fatalf("got %d, want -250", n.Minor())
	}
}

func TestString(t *testing.T) {
	usd, _ := FromMinor(12345, "USD")
	if usd.String() != "123.45 USD" {
		t.Fatalf("got %q", usd.String())
	}
	kwd, _ := FromMinor(12345, "KWD")
	if kwd.String() != "12.345 KWD" {
		t.Fatalf("got %q", kwd.String())
	}
	iqd, _ := FromMinor(5000, "IQD")
	if iqd.String() != "5000 IQD" {
		t.Fatalf("got %q", iqd.String())
	}
	neg, _ := FromMinor(-100, "USD")
	if neg.String() != "-1.00 USD" {
		t.Fatalf("got %q", neg.String())
	}
}

func TestRegister_Custom(t *testing.T) {
	if err := Register("ZZZ", 2); err != nil {
		t.Fatal(err)
	}
	m, err := New(1.50, "ZZZ")
	if err != nil {
		t.Fatal(err)
	}
	if m.Minor() != 150 {
		t.Fatal("custom currency wrong")
	}
}

func TestRegister_InvalidScale(t *testing.T) {
	if err := Register("AA", -1); err == nil {
		t.Fatal("expected error for negative scale")
	}
	if err := Register("AA", 9); err == nil {
		t.Fatal("expected error for scale > 8")
	}
}
