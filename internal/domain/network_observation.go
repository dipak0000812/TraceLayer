// internal/domain/network_observation.go
package domain

import (
	"errors"
	"fmt"
	"net/netip"
	"time"
)

// NetworkObservation is DATA_CONTRACT.md section 2.2. GeoCountry, ASN, and
// PropagationDelayMS are DERIVED — populated by enrichment/correlation, nil
// at raw parse time. Since Dipak owns GeoIP enrichment directly (no separate
// owner), that population happens in the same Go binary, not a teammate's
// package.
type NetworkObservation struct {
	ObservationID      string
	ObservedTXID       string
	ObservedAt         time.Time
	SrcIP              netip.Addr
	DstIP              netip.Addr
	SrcPort            uint16
	DstPort            uint16
	GeoCountry         *string
	ASN                *string
	PropagationDelayMS *int64
	CorrelationStatus  CorrelationStatus
	Provenance         Provenance
	DatasetID          string
	GeneratorVersion   string
}

var (
	ErrInvalidObservationID     = errors.New("domain: observation_id must match OBS-{24 hex chars}")
	ErrInvalidObservedTXID      = errors.New("domain: observed_txid must be 64 lowercase hex characters")
	ErrInvalidIP                = errors.New("domain: src_ip/dst_ip must be a valid IP address")
	ErrInvalidPort              = errors.New("domain: port must be in range 1024-65535")
	ErrInvalidCorrelationStatus = errors.New("domain: unrecognized correlation_status")
)

func (o NetworkObservation) Validate() error {
	if !ValidObservationID(o.ObservationID) {
		return fmt.Errorf("%w: %q", ErrInvalidObservationID, o.ObservationID)
	}
	if !ValidTXID(o.ObservedTXID) {
		return fmt.Errorf("%w: %q", ErrInvalidObservedTXID, o.ObservedTXID)
	}
	if !o.SrcIP.IsValid() || !o.DstIP.IsValid() {
		return ErrInvalidIP
	}
	if o.SrcPort < 1024 || o.DstPort < 1024 {
		return ErrInvalidPort
	}
	if !o.CorrelationStatus.Valid() {
		return fmt.Errorf("%w: %q", ErrInvalidCorrelationStatus, o.CorrelationStatus)
	}
	if !o.Provenance.Valid() {
		return fmt.Errorf("%w: %q", ErrInvalidProvenance, o.Provenance)
	}
	return nil
}