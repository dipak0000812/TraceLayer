// internal/evidence/evidence_test.go
package evidence

import (
	"context"
	"errors"
	"net/netip"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dipak0000812/TraceLayer/internal/correlation"
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

func insertTx(t *testing.T, db *storage.DB, txid string) {
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
		t.Fatalf("insertTx: %v", err)
	}
	t.Cleanup(func() {
		db.Pool().Exec(context.Background(), `DELETE FROM transactions WHERE txid = $1`, txid)
	})
}

func insertObsAt(t *testing.T, db *storage.DB, obsID, observedTXID string, observedAt time.Time) {
	t.Helper()
	obs := domain.NetworkObservation{
		ObservationID:     obsID,
		ObservedTXID:      observedTXID,
		ObservedAt:        observedAt,
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
		t.Fatalf("insertObsAt: %v", err)
	}
	t.Cleanup(func() {
		db.Pool().Exec(context.Background(), `DELETE FROM network_observations WHERE observation_id = $1`, obsID)
	})
}

func runCorrelation(t *testing.T, db *storage.DB) {
	t.Helper()
	if _, err := correlation.Run(context.Background(), db.Pool()); err != nil {
		t.Fatalf("correlation.Run: %v", err)
	}
}

func TestGetEvidence_TransactionNotFound(t *testing.T) {
	db := testDB(t)
	txid := strings.Repeat("5", 64)

	_, err := GetEvidence(context.Background(), db.Pool(), txid)
	if !errors.Is(err, storage.ErrTransactionNotFound) {
		t.Fatalf("expected ErrTransactionNotFound, got: %v", err)
	}
}

func TestGetEvidence_ZeroObservationsQualityZero(t *testing.T) {
	db := testDB(t)
	txid := strings.Repeat("6", 64)
	insertTx(t, db, txid)

	ev, err := GetEvidence(context.Background(), db.Pool(), txid)
	if err != nil {
		t.Fatalf("GetEvidence: %v", err)
	}
	if ev.NetworkQuality != 0 {
		t.Fatalf("NetworkQuality = %v, want 0 for zero observations", ev.NetworkQuality)
	}
	if len(ev.Observations) != 0 {
		t.Fatalf("expected 0 observations, got %d", len(ev.Observations))
	}
}

func TestGetEvidence_ThreeObservationsComputeExpectedQuality(t *testing.T) {
	db := testDB(t)
	txid := strings.Repeat("7", 64)
	insertTx(t, db, txid)

	base := time.Now().UTC().Truncate(time.Second)
	insertObsAt(t, db, "OBS-"+strings.Repeat("a", 24), txid, base)
	insertObsAt(t, db, "OBS-"+strings.Repeat("b", 24), txid, base.Add(10*time.Second))
	insertObsAt(t, db, "OBS-"+strings.Repeat("c", 24), txid, base.Add(20*time.Second))
	runCorrelation(t, db)

	ev, err := GetEvidence(context.Background(), db.Pool(), txid)
	if err != nil {
		t.Fatalf("GetEvidence: %v", err)
	}
	if len(ev.Observations) != 3 {
		t.Fatalf("expected 3 correlated observations, got %d", len(ev.Observations))
	}
	// N=3 -> count factor min(1,1)=1. spread=20s -> spread factor = 1-20/120.
	want := 1.0 * (1 - 20.0/120.0)
	if diff := ev.NetworkQuality - want; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("NetworkQuality = %v, want %v", ev.NetworkQuality, want)
	}
}
