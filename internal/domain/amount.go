// internal/domain/amount.go
package domain

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Satoshis is an exact Bitcoin amount: 1 BTC = 100,000,000 Satoshis.
// DATA_CONTRACT.md forbids storing monetary amounts as floating point.
// This type and ParseBTCString exist so that invariant is enforced by the
// type system, not by convention.
type Satoshis int64

const satoshisPerBTC = 100_000_000

var (
	ErrInvalidAmountFormat    = errors.New("domain: amount is not a valid decimal string")
	ErrAmountPrecisionOverflow = errors.New("domain: amount has more than 8 fractional digits")
	ErrNegativeAmount         = errors.New("domain: amount must not be negative")
)

// ParseBTCString converts a decimal BTC string as emitted by the seed-42
// generator (e.g. "0.74038447") into an exact Satoshis value using integer
// arithmetic only — never float parsing, which would round seed-42's
// 8-decimal-place values unpredictably.
func ParseBTCString(s string) (Satoshis, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ErrInvalidAmountFormat
	}
	if strings.HasPrefix(s, "-") {
		return 0, ErrNegativeAmount
	}

	whole, frac, hasFrac := strings.Cut(s, ".")
	if whole == "" {
		whole = "0"
	}
	if !isDigits(whole) {
		return 0, ErrInvalidAmountFormat
	}
	if hasFrac {
		if !isDigits(frac) {
			return 0, ErrInvalidAmountFormat
		}
		if len(frac) > 8 {
			return 0, ErrAmountPrecisionOverflow
		}
		frac += strings.Repeat("0", 8-len(frac))
	} else {
		frac = strings.Repeat("0", 8)
	}

	wholeVal, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, ErrInvalidAmountFormat
	}
	fracVal, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, ErrInvalidAmountFormat
	}
	return Satoshis(wholeVal*satoshisPerBTC + fracVal), nil
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// String renders Satoshis back as a BTC decimal string, for analyst-facing
// explanation text and error messages — not for persistence or re-parsing.
func (s Satoshis) String() string {
	if s < 0 {
		return "-" + Satoshis(-s).String()
	}
	whole := int64(s) / satoshisPerBTC
	frac := int64(s) % satoshisPerBTC
	return fmt.Sprintf("%d.%08d", whole, frac)
}