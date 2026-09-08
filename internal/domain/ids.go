// internal/domain/ids.go
package domain

import "regexp"

var (
	txidPattern         = regexp.MustCompile(`^[0-9a-f]{64}$`)
	observationIDPattern = regexp.MustCompile(`^OBS-[0-9a-f]{24}$`)
)

// ValidTXID reports whether s is exactly 64 lowercase hex characters, per
// API_CONTRACT.md section 1.7.
func ValidTXID(s string) bool { return txidPattern.MatchString(s) }

// ValidObservationID reports whether s matches OBS-{24 hex chars}, per
// API_CONTRACT.md section 1.7.
func ValidObservationID(s string) bool { return observationIDPattern.MatchString(s) }