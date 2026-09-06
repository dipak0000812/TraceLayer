// internal/domain/amount_test.go
package domain

import "testing"

func TestParseBTCString(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    Satoshis
		wantErr error
	}{
		{"seed42 input amount", "0.74038447", 74038447, nil},
		{"seed42 output amount", "0.73932792", 73932792, nil},
		{"seed42 fee", "0.00105655", 105655, nil},
		{"whole number", "1", 100000000, nil},
		{"no fractional part after dot", "2.", 0, ErrInvalidAmountFormat},
		{"empty string", "", 0, ErrInvalidAmountFormat},
		{"negative", "-0.1", 0, ErrNegativeAmount},
		{"garbage", "abc", 0, ErrInvalidAmountFormat},
		{"too much precision", "0.123456789", 0, ErrAmountPrecisionOverflow},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseBTCString(c.in)
			if c.wantErr != nil {
				if err != c.wantErr {
					t.Fatalf("ParseBTCString(%q) err = %v, want %v", c.in, err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseBTCString(%q) unexpected err = %v", c.in, err)
			}
			if got != c.want {
				t.Fatalf("ParseBTCString(%q) = %d, want %d", c.in, got, c.want)
			}
		})
	}
}

func TestSatoshisString_RoundTrips(t *testing.T) {
	s, err := ParseBTCString("0.74038447")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got, want := s.String(), "0.74038447"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}