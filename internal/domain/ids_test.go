// internal/domain/ids_test.go
package domain

import "testing"

func TestValidTXID(t *testing.T) {
	cases := map[string]bool{
		"b85038db8d34756615a8c82752b931ded57f99c1911097b5d0e21f0c648e638b": true, // seed-42 sample, 64 chars
		"":               false,
		"TOOSHORT":       false,
		"B85038DB8D34756615A8C82752B931DED57F99C1911097B5D0E21F0C648E638B": false, // uppercase rejected
	}
	for in, want := range cases {
		if got := ValidTXID(in); got != want {
			t.Errorf("ValidTXID(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestValidObservationID(t *testing.T) {
	cases := map[string]bool{
		"OBS-e93a6d2b75347fe43e0458c6": true, // seed-42 sample, 24 hex chars after prefix
		"OBS-tooshort":                 false,
		"e93a6d2b75347fe43e0458c6":     false, // missing prefix
	}
	for in, want := range cases {
		if got := ValidObservationID(in); got != want {
			t.Errorf("ValidObservationID(%q) = %v, want %v", in, got, want)
		}
	}
}