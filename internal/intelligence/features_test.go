// internal/intelligence/features_test.go
package intelligence

import (
	"net/netip"
	"testing"
	"time"

	"github.com/dipak0000812/TraceLayer/internal/domain"
	"github.com/dipak0000812/TraceLayer/internal/evidence"
)

func TestBuildFeatures_MapsEvidenceCorrectly(t *testing.T) {
	geoCountry := "BR"
	ev := evidence.Evidence{
		Transaction: domain.Transaction{
			TXID:            "b85038db8d34756615a8c82752b931ded57f99c1911097b5d0e21f0c648e638b",
			InputAddresses:  []string{"sbc1input"},
			OutputAddresses: []string{"sbc1output"},
			InputAmounts:    []domain.Satoshis{74038447},
			OutputAmounts:   []domain.Satoshis{73932792},
			Fee:             105655,
			ScriptType:      domain.ScriptTypeP2PKH,
		},
		Observations: []domain.NetworkObservation{{
			ObservationID: "OBS-feature-test",
			ObservedAt:    time.Date(2025, 3, 22, 16, 34, 24, 0, time.UTC),
			SrcIP:         netip.MustParseAddr("172.16.2.66"),
			GeoCountry:    &geoCountry,
		}},
	}

	f := BuildFeatures(ev)
	if f.TXID != ev.Transaction.TXID {
		t.Errorf("TXID mismatch")
	}
	if f.AmountBTC != 0.73932792 {
		t.Errorf("AmountBTC = %v, want 0.73932792", f.AmountBTC)
	}
	if f.FeeBTC != 0.00105655 {
		t.Errorf("FeeBTC = %v, want 0.00105655", f.FeeBTC)
	}
	if len(f.NetworkObservations) != 1 || f.NetworkObservations[0].SrcIP != "172.16.2.66" {
		t.Fatalf("network observation not mapped correctly: %+v", f.NetworkObservations)
	}
	if f.NetworkObservations[0].ObservationID != "OBS-feature-test" {
		t.Errorf("observation ID was not mapped")
	}
	if len(f.InputAmounts) != 1 || f.InputAmounts[0] != 0.74038447 {
		t.Errorf("InputAmounts = %v, want [0.74038447]", f.InputAmounts)
	}
	if len(f.OutputAmounts) != 1 || f.OutputAmounts[0] != 0.73932792 {
		t.Errorf("OutputAmounts = %v, want [0.73932792]", f.OutputAmounts)
	}
}
