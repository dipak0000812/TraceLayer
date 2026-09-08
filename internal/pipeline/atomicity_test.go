package pipeline

import (
	"context"
	"net/netip"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dipak0000812/TraceLayer/internal/correlation"
	"github.com/dipak0000812/TraceLayer/internal/domain"
	"github.com/dipak0000812/TraceLayer/internal/entity"
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

func truncateAll(t *testing.T, db *storage.DB) {
	t.Helper()
	tables := []string{"forensic_leads", "entities", "network_observations", "transactions"}
	for _, tbl := range tables {
		if _, err := db.Pool().Exec(context.Background(), "TRUNCATE TABLE "+tbl+" CASCADE"); err != nil {
			t.Fatalf("truncating %s: %v", tbl, err)
		}
	}
}

func insertTxWithInputs(t *testing.T, db *storage.DB, txid string, inputAddrs []string) {
	t.Helper()
	amounts := make([]domain.Satoshis, len(inputAddrs))
	for i := range amounts {
		amounts[i] = 1000
	}
	tx := domain.Transaction{
		TXID:             txid,
		BlockTime:        time.Now().UTC().Truncate(time.Second),
		InputAddresses:   inputAddrs,
		OutputAddresses:  []string{"sbc1output"},
		InputAmounts:     amounts,
		OutputAmounts:    []domain.Satoshis{900},
		Fee:              100,
		ScriptType:       domain.ScriptTypeP2PKH,
		Provenance:       domain.ProvenanceSynthetic,
		DatasetID:        "6bc084b677a63411",
		GeneratorVersion: "1.0.0",
	}
	if _, err := storage.InsertTransaction(context.Background(), db.Pool(), tx); err != nil {
		t.Fatalf("insertTxWithInputs: %v", err)
	}
}

func insertObsFor(t *testing.T, db *storage.DB, obsID, observedTXID string) {
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
		t.Fatalf("insertObsFor: %v", err)
	}
}

// TestCorrelateAtomicity_PartialFailureLeavesPriorStepsCommitted documents the
// current gap: correlation writes are committed before a later entity step
// fails because the orchestration does not share a database transaction.
func TestCorrelateAtomicity_PartialFailureLeavesPriorStepsCommitted(t *testing.T) {
	db := testDB(t)
	truncateAll(t, db)
	t.Cleanup(func() { truncateAll(t, db) })

	txid := strings.Repeat("a", 64)
	insertTxWithInputs(t, db, txid, []string{"sbc1atomicA", "sbc1atomicB"})
	insertObsFor(t, db, "OBS-"+strings.Repeat("f", 24), txid)

	corrResult, err := correlation.Run(context.Background(), db.Pool())
	if err != nil {
		t.Fatalf("correlation.Run: %v", err)
	}
	if corrResult.Correlated != 1 {
		t.Fatalf("expected 1 correlated observation, got %d", corrResult.Correlated)
	}

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := entity.Run(canceledCtx, db.Pool(), domain.ProvenanceDerived, "6bc084b677a63411"); err == nil {
		t.Fatalf("expected entity.Run to fail against a pre-canceled context")
	}

	var status string
	err = db.Pool().QueryRow(context.Background(),
		`SELECT correlation_status FROM network_observations WHERE observed_txid = $1`, txid).Scan(&status)
	if err != nil {
		t.Fatalf("fetching observation status: %v", err)
	}
	if status != string(domain.CorrelationCorrelated) {
		t.Fatalf("status = %s — rollback appears to have been added; update this test to assert rollback", status)
	}
}
