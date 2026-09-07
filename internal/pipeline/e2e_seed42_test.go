// internal/pipeline/e2e_seed42_test.go
package pipeline

import (
	"context"
	"os"
	"testing"

	"github.com/dipak0000812/TraceLayer/internal/correlation"
	"github.com/dipak0000812/TraceLayer/internal/domain"
	"github.com/dipak0000812/TraceLayer/internal/entity"
	"github.com/dipak0000812/TraceLayer/internal/ingestion"
	"github.com/dipak0000812/TraceLayer/internal/ranking"
	"github.com/dipak0000812/TraceLayer/internal/storage"
)

const (
	seed42TxCSVPath  = "../../data/synthetic/datasets/seed-42/data/raw/transactions.csv"
	seed42NetCSVPath = "../../data/synthetic/datasets/seed-42/data/raw/network_observations.csv"

	wantTxTotal     = 109
	wantTxIngested  = 106
	wantTxDuplicate = 3
	wantNetTotal    = 368
	wantEntities    = 59
	wantCorrelated  = 368
	wantLeads       = 59
)

// TestE2E_Seed42_FullPipeline runs the real seed-42 CSVs through
// ingest -> ingest -> correlate -> entity cluster -> rank, asserting the
// exact counts confirmed against this dataset across multiple manual runs
// this session (Phase 7, Phase 10, Phase 12). If any of these numbers
// change, that's either a real regression or the dataset itself changed —
// either way, don't update the constants above without re-confirming the
// new numbers the same way these were confirmed, not by just making the
// test pass.
func TestE2E_Seed42_FullPipeline(t *testing.T) {
	db := testDB(t)
	truncateAll(t, db)
	t.Cleanup(func() { truncateAll(t, db) })

	txFile, err := os.Open(seed42TxCSVPath)
	if err != nil {
		t.Fatalf("opening %s: %v (run from repo root or adjust the relative path for your test runner)", seed42TxCSVPath, err)
	}
	defer txFile.Close()

	validTx, rejectedTx, err := ingestion.ParseTransactionsCSV(txFile)
	if err != nil {
		t.Fatalf("ParseTransactionsCSV: %v", err)
	}
	if len(validTx)+len(rejectedTx) != wantTxTotal {
		t.Fatalf("parsed %d valid + %d rejected = %d rows, want %d total rows in seed-42 transactions.csv",
			len(validTx), len(rejectedTx), len(validTx)+len(rejectedTx), wantTxTotal)
	}
	if len(rejectedTx) != 0 {
		t.Fatalf("expected 0 rejected transaction rows in seed-42 (all should be structurally valid), got %d: %+v",
			len(rejectedTx), rejectedTx)
	}

	ingested, duplicate := 0, 0
	for _, tx := range validTx {
		result, err := storage.InsertTransaction(context.Background(), db.Pool(), tx)
		if err != nil {
			t.Fatalf("InsertTransaction(%s): %v", tx.TXID, err)
		}
		switch result {
		case storage.Inserted:
			ingested++
		case storage.DuplicateNoop:
			duplicate++
		case storage.Conflict:
			t.Fatalf("txid %s reported CONFLICT — seed-42's known duplicates should be identical-payload no-ops, not conflicts", tx.TXID)
		}
	}
	if ingested != wantTxIngested {
		t.Fatalf("ingested_count = %d, want %d", ingested, wantTxIngested)
	}
	if duplicate != wantTxDuplicate {
		t.Fatalf("duplicate_count = %d, want %d", duplicate, wantTxDuplicate)
	}

	netFile, err := os.Open(seed42NetCSVPath)
	if err != nil {
		t.Fatalf("opening %s: %v", seed42NetCSVPath, err)
	}
	defer netFile.Close()

	validObs, rejectedObs, err := ingestion.ParseNetworkObservationsCSV(netFile)
	if err != nil {
		t.Fatalf("ParseNetworkObservationsCSV: %v", err)
	}
	if len(validObs)+len(rejectedObs) != wantNetTotal {
		t.Fatalf("parsed %d valid + %d rejected = %d rows, want %d total rows in seed-42 network_observations.csv",
			len(validObs), len(rejectedObs), len(validObs)+len(rejectedObs), wantNetTotal)
	}
	if len(rejectedObs) != 0 {
		t.Fatalf("expected 0 rejected observation rows in seed-42, got %d: %+v", len(rejectedObs), rejectedObs)
	}

	for _, obs := range validObs {
		if _, err := storage.InsertNetworkObservation(context.Background(), db.Pool(), obs); err != nil {
			t.Fatalf("InsertNetworkObservation(%s): %v", obs.ObservationID, err)
		}
	}

	corrResult, err := correlation.Run(context.Background(), db.Pool())
	if err != nil {
		t.Fatalf("correlation.Run: %v", err)
	}
	if corrResult.Correlated != wantCorrelated {
		t.Fatalf("observations_correlated = %d, want %d", corrResult.Correlated, wantCorrelated)
	}
	if corrResult.Orphaned != 0 {
		t.Fatalf("orphaned = %d, want 0 — every seed-42 observation should reference an ingested txid", corrResult.Orphaned)
	}

	entityResult, err := entity.Run(context.Background(), db.Pool(), domain.ProvenanceDerived, "6bc084b677a63411")
	if err != nil {
		t.Fatalf("entity.Run: %v", err)
	}
	if entityResult.EntitiesClustered != wantEntities {
		t.Fatalf("entities_clustered = %d, want %d", entityResult.EntitiesClustered, wantEntities)
	}

	leadsCount, err := ranking.Run(context.Background(), db.Pool(), nil)
	if err != nil {
		t.Fatalf("ranking.Run: %v", err)
	}
	if leadsCount != wantLeads {
		t.Fatalf("ranking.Run returned %d leads, want %d", leadsCount, wantLeads)
	}

	var leadCount int
	err = db.Pool().QueryRow(context.Background(), `SELECT count(*) FROM forensic_leads`).Scan(&leadCount)
	if err != nil {
		t.Fatalf("counting forensic_leads: %v", err)
	}
	if leadCount != wantLeads {
		t.Fatalf("forensic_leads row count = %d, want %d — mismatch vs ranking.Run's own return value (%d) would mean the function's count and its actual writes disagree", leadCount, wantLeads, leadsCount)
	}
}
