// internal/entity/run_test.go
package entity

import (
	"context"
	"os"
	"sort"
	"strings"
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
	t.Cleanup(func() {
		db.Pool().Exec(context.Background(), `DELETE FROM transactions WHERE txid = $1`, txid)
	})
}

func cleanupEntity(t *testing.T, db *storage.DB, entityID string) {
	t.Helper()
	t.Cleanup(func() {
		db.Pool().Exec(context.Background(), `DELETE FROM entities WHERE entity_id = $1`, entityID)
	})
}

func expectedIDFor(addrs []string) string {
	sorted := append([]string{}, addrs...)
	sort.Strings(sorted)
	return deterministicEntityID(sorted)
}

func TestRun_ClustersAndPersistsEntity(t *testing.T) {
	db := testDB(t)
	txid := strings.Repeat("8", 64)
	addrs := []string{"sbc1runA", "sbc1runB"}
	insertTxWithInputs(t, db, txid, addrs)

	expectedID := expectedIDFor(addrs)
	cleanupEntity(t, db, expectedID)

	result, err := Run(context.Background(), db.Pool(), domain.ProvenanceSynthetic, "6bc084b677a63411")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.EntitiesClustered < 1 {
		t.Fatalf("expected at least 1 entity clustered, got %d", result.EntitiesClustered)
	}

	var clusterSize int
	err = db.Pool().QueryRow(context.Background(),
		`SELECT cluster_size FROM entities WHERE entity_id = $1`, expectedID).Scan(&clusterSize)
	if err != nil {
		t.Fatalf("fetching persisted entity: %v", err)
	}
	if clusterSize != 2 {
		t.Fatalf("cluster_size = %d, want 2", clusterSize)
	}
}

func TestRun_IdempotentOnReRun(t *testing.T) {
	db := testDB(t)
	txid := strings.Repeat("9", 64)
	addrs := []string{"sbc1idemA", "sbc1idemB"}
	insertTxWithInputs(t, db, txid, addrs)
	expectedID := expectedIDFor(addrs)
	cleanupEntity(t, db, expectedID)

	if _, err := Run(context.Background(), db.Pool(), domain.ProvenanceSynthetic, "6bc084b677a63411"); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	if _, err := Run(context.Background(), db.Pool(), domain.ProvenanceSynthetic, "6bc084b677a63411"); err != nil {
		t.Fatalf("second Run: %v", err)
	}

	var count int
	err := db.Pool().QueryRow(context.Background(),
		`SELECT count(*) FROM entities WHERE entity_id = $1`, expectedID).Scan(&count)
	if err != nil {
		t.Fatalf("counting entity rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("entity row count = %d after two runs, want 1 (idempotent)", count)
	}
}