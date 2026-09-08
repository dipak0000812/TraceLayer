package storage 

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/dipak0000812/TraceLayer/internal/domain"
)

func sampleObs(id string) domain.NetworkObservation {
	return domain.NetworkObservation{
		ObservationID:     id,
		ObservedTXID:      "b85038db8d34756615a8c82752b931ded57f99c1911097b5d0e21f0c648e638b",
		ObservedAt:        time.Now().UTC().Truncate(time.Second),
		SrcIP:             netip.MustParseAddr("203.0.113.10"),
		DstIP:             netip.MustParseAddr("198.51.100.20"),
		SrcPort:           18333,
		DstPort:           8333,
		CorrelationStatus: domain.CorrelationPending,
		Provenance:        domain.ProvenanceSynthetic,
		DatasetID:         "6bc084b677a63411",
		GeneratorVersion:  "1.0.0",
	}
}

func cleanupObservation(t *testing.T, db *DB, observationID string) {
	t.Helper()
	t.Cleanup(func() {
		_, err := db.Pool().Exec(context.Background(), `DELETE FROM network_observations WHERE observation_id = $1`, observationID)
		if err != nil {
			t.Logf("cleanup: failed to delete observation ID %s: %v", observationID, err)
		}
	})
}

func TestInsertNetworkObservation_FreshInsert(t *testing.T) {
	db := testPool(t)
	observationID := "OBS-eeeeeeeeeeeeeeeeeeeeeeee"
	cleanupObservation(t, db, observationID)
	result, err := InsertNetworkObservation(context.Background(), db.Pool(), sampleObs(observationID))
	if err != nil {
		t.Fatalf("InsertNetworkObservation: %v", err)
	}
	if result != NetworkInserted {
		t.Fatalf("result = %s, want INSERTED", result)
	}
}

func TestInsertNetworkObservation_DuplicateObservationIDRejected(t *testing.T) {
	db := testPool(t)
	ctx := context.Background()
	observationID := "OBS-ffffffffffffffffffffffff"
	cleanupObservation(t, db, observationID)
	obs := sampleObs(observationID)

	if _, err := InsertNetworkObservation(ctx, db.Pool(), obs); err != nil {
		t.Fatalf("first insert: %v", err)
	}

	changed := obs
	changed.ObservedTXID = "9e752d5ae640e33b1ea420bb782e7b59617465516801b8d0599378e4694917d2"
	result, err := InsertNetworkObservation(ctx, db.Pool(), changed)
	if err != nil {
		t.Fatalf("second insert: %v", err)
	}
	if result != NetworkDuplicate {
		t.Fatalf("result = %s, want DUPLICATE", result)
	}

	var storedTXID string
	err = db.Pool().QueryRow(ctx, `SELECT observed_txid FROM network_observations WHERE observation_id = $1`, obs.ObservationID).Scan(&storedTXID)
	if err != nil {
		t.Fatalf("fetching stored row: %v", err)
	}
	if storedTXID != obs.ObservedTXID {
		t.Fatalf("existing row was overwritten: observed_txid = %s, want unchanged %s", storedTXID, obs.ObservedTXID)
	}
}