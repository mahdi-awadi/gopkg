// Package money provides an immutable Money type with explicit currency
// and minor-unit precision.
//
// Money is stored as int64 minor units (e.g. cents for USD, fils for KWD).
// Currency precision is consulted via the Currency type; the package ships
// ISO 4217 defaults for the most common codes and lets consumers register
// extras.
//
// No floats ever. Arithmetic across mismatched currencies returns an error.
//
// Zero third-party deps.
package money

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
)

// Errors.
var (
	ErrCurrencyMismatch = errors.New("money: currency mismatch")
	ErrDivideByZero     = errors.New("money: divide by zero")
	ErrOverflow         = errors.New("money: overflow")
	ErrUnknownCurrency  = errors.New("money: unknown currency")
	ErrInvalidFormat    = errors.New("money: invalid format")
)

// Currency is an ISO 4217 code (e.g. "USD", "KWD", "IQD", "EUR").
type Currency string

// Code returns the uppercase ISO code.
func (c Currency) Code() string { return strings.ToUpper(string(c)) }

// registry maps Currency → minor-unit scale (number of decimal places).
var (
	registryMu sync.RWMutex
	registry   = map[string]int{
		// ISO 4217 defaults — commonly used subset.
		"USD": 2, "EUR": 2, "GBP": 2, "CAD": 2, "AUD": 2, "CHF": 2, "NZD": 2,
		"SEK": 2, "NOK": 2, "DKK": 2, "PLN": 2, "CZK": 2, "HUF": 2, "RON": 2,
		"SGD": 2, "HKD": 2, "TWD": 2, "CNY": 2, "INR": 2, "THB": 2, "MYR": 2, "IDR": 2,
		"JPY": 0, "KRW": 0, "VND": 0, "CLP": 0, "ISK": 0,
		"IQD": 0, // practical precision for Iraqi dinar prices
		"KWD": 3, "BHD": 3, "OMR": 3, "JOD": 3, "TND": 3, // 3-decimal currencies
		"AED": 2, "SAR": 2, "EGP": 2, "QAR": 2, "LBP": 2, "TRY": 2, "ILS": 2,
		"ZAR": 2, "NGN": 2, "KES": 2, "MAD": 2, "DZD": 2,
		"BRL": 2, "MXN": 2, "ARS": 2, "COP": 2, "PEN": 2,
		"RUB": 2,
	}
)

// Register adds or overrides a currency's minor-unit scale.
// Returns an error if scale is negative or greater than 8.
func Register(code string, scale int) error {
	if scale < 0 || scale > 8 {
		return fmt.Errorf("money: scale must be in [0, 8]")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[strings.ToUpper(code)] = scale
	return nil
}

// Scale returns the number of decimal places for the currency, or an error
// if the currency is unregistered.
func Scale(c Currency) (int, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	s, ok := registry[c.Code()]
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrUnknownCurrency, c)
	}
	return s, nil
}

// Money is an immutable amount-in-currency. Zero value is $0.00 USD-scale-0
// which is not useful — construct via New / FromMinor.
type Money struct {
	minor    int64
	currency Currency
}

// New creates a Money value from amount in **major units** (e.g. dollars).
// Returns an error if the currency is unregistered.
// Precision beyond the currency's scale is truncated toward zero.
func New(amount float64, currency Currency) (Money, error) {
	scale, err := Scale(currency)
	if err != nil {
		return Money{}, err
	}
	factor := math.Pow10(scale)
	// Guard against NaN/Inf/overflow.
	if math.IsNaN(amount) || math.IsInf(amount, 0) {
		return Money{}, ErrInvalidFormat
	}
	v := amount * factor
	if v > math.MaxInt64 || v < math.MinInt64 {
		return Money{}, ErrOverflow
	}
	return Money{minor: int64(v), currency: currency}, nil
}

// FromMinor creates a Money directly from the minor-unit integer.
// (e.g. FromMinor(150, "USD") → $1.50)
func FromMinor(minor int64, currency Currency) (Money, error) {
	if _, err := Scale(currency); err != nil {
		return Money{}, err
	}
	return Money{minor: minor, currency: currency}, nil
}

// Minor returns the amount in minor units.
func (m Money) Minor() int64 { return m.minor }

// Currency returns the Currency.
func (m Money) Currency() Currency { return m.currency }

// IsZero reports whether the amount is zero.
func (m Money) IsZero() bool { return m.minor == 0 }

// Add returns m + other (same currency). Returns ErrCurrencyMismatch on mismatch.
func (m Money) Add(other Money) (Money, error) {
	if err := m.sameCurrency(other); err != nil {
		return Money{}, err
	}
	if addOverflows(m.minor, other.minor) {
		return Money{}, ErrOverflow
	}
	return Money{minor: m.minor + other.minor, currency: m.currency}, nil
}

// Sub returns m - other (same currency).
func (m Money) Sub(other Money) (Money, error) {
	if err := m.sameCurrency(other); err != nil {
		return Money{}, err
	}
	if subOverflows(m.minor, other.minor) {
		return Money{}, ErrOverflow
	}
	return Money{minor: m.minor - other.minor, currency: m.currency}, nil
}

// MulInt returns m * factor (integer multiplier, no precision loss).
func (m Money) MulInt(factor int64) (Money, error) {
	if factor == 0 {
		return Money{minor: 0, currency: m.currency}, nil
	}
	if m.minor != 0 {
		if (m.minor*factor)/factor != m.minor {
			return Money{}, ErrOverflow
		}
	}
	return Money{minor: m.minor * factor, currency: m.currency}, nil
}

// DivInt returns m / divisor rounded toward zero (integer division).
func (m Money) DivInt(divisor int64) (Money, error) {
	if divisor == 0 {
		return Money{}, ErrDivideByZero
	}
	return Money{minor: m.minor / divisor, currency: m.currency}, nil
}

// Equal reports whether m and other share currency and amount.
func (m Money) Equal(other Money) bool {
	return m.currency.Code() == other.currency.Code() && m.minor == other.minor
}

// Cmp returns -1/0/+1 if m is less/equal/greater than other. Returns
// ErrCurrencyMismatch if currencies differ.
func (m Money) Cmp(other Money) (int, error) {
	if err := m.sameCurrency(other); err != nil {
		return 0, err
	}
	switch {
	case m.minor < other.minor:
		return -1, nil
	case m.minor > other.minor:
		return 1, nil
	}
	return 0, nil
}

// Negate returns -m.
func (m Money) Negate() Money {
	return Money{minor: -m.minor, currency: m.currency}
}

// String renders the amount with appropriate decimals (e.g. "$1.50" for
// USD; "0.500 KWD" for KWD; "150 IQD" for IQD). The symbol is the ISO code.
func (m Money) String() string {
	scale, err := Scale(m.currency)
	if err != nil {
		return fmt.Sprintf("%d <unknown:%s>", m.minor, m.currency)
	}
	negative := m.minor < 0
	abs := m.minor
	if negative {
		abs = -abs
	}
	if scale == 0 {
		if negative {
			return fmt.Sprintf("-%d %s", abs, m.currency.Code())
		}
		return fmt.Sprintf("%d %s", abs, m.currency.Code())
	}
	divisor := int64(math.Pow10(scale))
	whole := abs / divisor
	frac := abs % divisor
	fracStr := strconv.FormatInt(frac, 10)
	if len(fracStr) < scale {
		fracStr = strings.Repeat("0", scale-len(fracStr)) + fracStr
	}
	sign := ""
	if negative {
		sign = "-"
	}
	return fmt.Sprintf("%s%d.%s %s", sign, whole, fracStr, m.currency.Code())
}

func (m Money) sameCurrency(other Money) error {
	if m.currency.Code() != other.currency.Code() {
		return fmt.Errorf("%w: %s vs %s", ErrCurrencyMismatch, m.currency, other.currency)
	}
	return nil
}

func addOverflows(a, b int64) bool {
	return (b > 0 && a > math.MaxInt64-b) || (b < 0 && a < math.MinInt64-b)
}

func subOverflows(a, b int64) bool {
	return (b < 0 && a > math.MaxInt64+b) || (b > 0 && a < math.MinInt64+b)
}
