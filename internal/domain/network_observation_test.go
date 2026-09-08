// internal/domain/network_observation_test.go
package domain

import (
	"errors"
	"net/netip"
	"testing"
	"time"
)

func validSeed42Obs() NetworkObservation {
	// Exact values from the seed-42 sample row in network_observations.csv.
	return NetworkObservation{
		ObservationID:     "OBS-e93a6d2b75347fe43e0458c6",
		ObservedTXID:      "b85038db8d34756615a8c82752b931ded57f99c1911097b5d0e21f0c648e638b",
		ObservedAt:        time.Now(),
		SrcIP:             netip.MustParseAddr("198.51.100.98"),
		DstIP:             netip.MustParseAddr("203.0.113.8"),
		SrcPort:           15749,
		DstPort:           15780,
		CorrelationStatus: CorrelationPending,
		Provenance:        ProvenanceSynthetic,
		DatasetID:         "6bc084b677a63411",
		GeneratorVersion:  "1.0.0",
	}
}

func TestNetworkObservation_Validate_Valid(t *testing.T) {
	if err := validSeed42Obs().Validate(); err != nil {
		t.Fatalf("expected valid seed-42 observation, got err: %v", err)
	}
}

func TestNetworkObservation_Validate_BadObservationID(t *testing.T) {
	o := validSeed42Obs()
	o.ObservationID = "not-an-id"
	if err := o.Validate(); !errors.Is(err, ErrInvalidObservationID) {
		t.Fatalf("expected ErrInvalidObservationID, got: %v", err)
	}
}

func TestNetworkObservation_Validate_PortTooLow(t *testing.T) {
	o := validSeed42Obs()
	o.SrcPort = 80 // below the 1024 floor DATA_CONTRACT.md specifies
	if err := o.Validate(); !errors.Is(err, ErrInvalidPort) {
		t.Fatalf("expected ErrInvalidPort, got: %v", err)
	}
}