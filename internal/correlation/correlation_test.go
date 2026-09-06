// internal/correlation/correlation_test.go
package correlation

import (
	"context"
	"net/netip"
	"os"
	"testing"
	"time"

	"github.com/dipak0000812/TraceLayer/internal/domain"
	"github.com/dipak0000812/TraceLayer/internal/storage"
)

func testDB(t *testing.T) *storage.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping live Postgres test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db, err := storage.New(ctx, storage.Config{DSN: dsn, ConnectTimeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("storage.New(): %v", err)
	}
	t.Cleanup(db.Close)
	return db
}

func insertSampleTx(t *testing.T, db *storage.DB, txid string) {
	t.Helper()
	tx := domain.Transaction{
		TXID:             txid,
		BlockTime:        time.Now().UTC().Truncate(time.Second),
		InputAddresses:   []string{"sbc1input"},
		OutputAddresses:  []string{"sbc1output"},
		InputAmounts:     []domain.Satoshis{1000},
		OutputAmounts:    []domain.Satoshis{900},
		Fee:              100,
		ScriptType:       domain.ScriptTypeP2PKH,
		Provenance:       domain.ProvenanceSynthetic,
		DatasetID:        "6bc084b677a63411",
		GeneratorVersion: "1.0.0",
	}
	if _, err := storage.InsertTransaction(context.Background(), db.Pool(), tx); err != nil {
		t.Fatalf("insertSampleTx: %v", err)
	}
	t.Cleanup(func() {
		db.Pool().Exec(context.Background(), `DELETE FROM transactions WHERE txid = $1`, txid)
	})
}

func insertSampleObs(t *testing.T, db *storage.DB, obsID, observedTXID string) {
	t.Helper()
	obs := domain.NetworkObservation{
		ObservationID:     obsID,
		ObservedTXID:      observedTXID,
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
	if _, err := storage.InsertNetworkObservation(context.Background(), db.Pool(), obs); err != nil {
		t.Fatalf("insertSampleObs: %v", err)
	}
	t.Cleanup(func() {
		db.Pool().Exec(context.Background(), `DELETE FROM network_observations WHERE observation_id = $1`, obsID)
	})
}

func fetchStatus(t *testing.T, db *storage.DB, obsID string) domain.CorrelationStatus {
	t.Helper()
	var status string
	err := db.Pool().QueryRow(context.Background(),
		`SELECT correlation_status FROM network_observations WHERE observation_id = $1`, obsID).Scan(&status)
	if err != nil {
		t.Fatalf("fetchStatus: %v", err)
	}
	return domain.CorrelationStatus(status)
}

func TestRun_CorrelatesWhenMatchingTransactionExists(t *testing.T) {
	db := testDB(t)
	txid := "1111111111111111111111111111111111111111111111111111111111111111"[:64]
	insertSampleTx(t, db, txid)
	insertSampleObs(t, db, "OBS-111111111111111111111111", txid)

	result, err := Run(context.Background(), db.Pool())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Correlated < 1 {
		t.Fatalf("expected at least 1 correlated row, got %d", result.Correlated)
	}
	if got := fetchStatus(t, db, "OBS-111111111111111111111111"); got != domain.CorrelationCorrelated {
		t.Fatalf("status = %s, want CORRELATED", got)
	}
}

func TestRun_MarksOrphanWhenNoMatchingTransaction(t *testing.T) {
	db := testDB(t)
	nonexistentTXID := "2222222222222222222222222222222222222222222222222222222222222222"[:64]
	insertSampleObs(t, db, "OBS-222222222222222222222222", nonexistentTXID)

	result, err := Run(context.Background(), db.Pool())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Orphaned < 1 {
		t.Fatalf("expected at least 1 orphaned row, got %d", result.Orphaned)
	}
	if got := fetchStatus(t, db, "OBS-222222222222222222222222"); got != domain.CorrelationOrphan {
		t.Fatalf("status = %s, want ORPHAN", got)
	}
}

// This is the test that actually proves the design decision you made:
// an ORPHAN observation must correlate once its transaction shows up later,
// not stay stuck ORPHAN forever from a single evaluation.
func TestRun_OrphanRetriedAfterTransactionArrivesLater(t *testing.T) {
	db := testDB(t)
	txid := "3333333333333333333333333333333333333333333333333333333333333333"[:64]
	insertSampleObs(t, db, "OBS-333333333333333333333333", txid)

	if _, err := Run(context.Background(), db.Pool()); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	if got := fetchStatus(t, db, "OBS-333333333333333333333333"); got != domain.CorrelationOrphan {
		t.Fatalf("after first run, status = %s, want ORPHAN", got)
	}

	insertSampleTx(t, db, txid)

	if _, err := Run(context.Background(), db.Pool()); err != nil {
		t.Fatalf("second Run: %v", err)
	}
	if got := fetchStatus(t, db, "OBS-333333333333333333333333"); got != domain.CorrelationCorrelated {
		t.Fatalf("after second run, status = %s, want CORRELATED — ORPHAN must be retryable", got)
	}
}

func TestRun_CorrelatedRowsNeverReevaluated(t *testing.T) {
	db := testDB(t)
	txid := "4444444444444444444444444444444444444444444444444444444444444444"[:64]
	insertSampleTx(t, db, txid)
	insertSampleObs(t, db, "OBS-444444444444444444444444", txid)

	if _, err := Run(context.Background(), db.Pool()); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	if got := fetchStatus(t, db, "OBS-444444444444444444444444"); got != domain.CorrelationCorrelated {
		t.Fatalf("status = %s, want CORRELATED", got)
	}

	result, err := Run(context.Background(), db.Pool())
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	// This specific row is CORRELATED and terminal — it must not appear in
	// either count on a re-run. (Other tests' rows may still be transiently
	// PENDING/ORPHAN if run concurrently, so this asserts the row directly
	// rather than asserting global RunResult counts are zero.)
	if got := fetchStatus(t, db, "OBS-444444444444444444444444"); got != domain.CorrelationCorrelated {
		t.Fatalf("status changed on re-run: %s, CORRELATED must be terminal", got)
	}
	_ = result
}