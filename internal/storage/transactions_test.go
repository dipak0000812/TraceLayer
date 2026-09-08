// internal/storage/transactions_test.go
package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/dipak0000812/TraceLayer/internal/domain"
)

func testPool(t *testing.T) *DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping live Postgres test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db, err := New(ctx, Config{DSN: dsn, ConnectTimeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("New(): %v", err)
	}
	t.Cleanup(db.Close)
	return db
}

func sampleTx(txid string) domain.Transaction {
	return domain.Transaction{
		TXID:             txid,
		BlockTime:        time.Now().UTC().Truncate(time.Second),
		InputAddresses:   []string{"sbc1input"},
		OutputAddresses:  []string{"sbc1output"},
		InputAmounts:     []domain.Satoshis{74038447},
		OutputAmounts:    []domain.Satoshis{73932792},
		Fee:              105655,
		ScriptType:       domain.ScriptTypeP2PKH,
		Provenance:       domain.ProvenanceSynthetic,
		DatasetID:        "6bc084b677a63411",
		GeneratorVersion: "1.0.0",
	}
}

func cleanupTx(t *testing.T, db *DB, txid string) {
	t.Helper()
	t.Cleanup(func() {
		_, err := db.Pool().Exec(context.Background(), `DELETE FROM transactions WHERE txid = $1`, txid)
		if err != nil {
			t.Logf("cleanup: failed to delete txid %s: %v", txid, err)
		}
	})
}

func TestInsertTransaction_FreshInsert(t *testing.T) {
	db := testPool(t)
	txid := "aaaa000000000000000000000000000000000000000000000000000000aaaa"
	cleanupTx(t, db, txid)
	result, err := InsertTransaction(context.Background(), db.Pool(),
		sampleTx(txid))
	if err != nil {
		t.Fatalf("InsertTransaction: %v", err)
	}
	if result != Inserted {
		t.Fatalf("result = %s, want INSERTED", result)
	}
}

func TestInsertTransaction_IdenticalPayloadIsNoop(t *testing.T) {
	db := testPool(t)
	txid := "bbbb000000000000000000000000000000000000000000000000000000bbbb"
	cleanupTx(t, db, txid)
	ctx := context.Background()
	tx := sampleTx(txid)

	if _, err := InsertTransaction(ctx, db.Pool(), tx); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	result, err := InsertTransaction(ctx, db.Pool(), tx)
	if err != nil {
		t.Fatalf("second insert: %v", err)
	}
	if result != DuplicateNoop {
		t.Fatalf("result = %s, want DUPLICATE_NOOP", result)
	}
}

func TestInsertTransaction_ConflictingPayloadDetected(t *testing.T) {
	db := testPool(t)
	ctx := context.Background()
	txid := "cccc000000000000000000000000000000000000000000000000000000cccc"
	cleanupTx(t, db, txid)

	if _, err := InsertTransaction(ctx, db.Pool(), sampleTx(txid)); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	changed := sampleTx(txid)
	changed.Fee = 999999
	result, err := InsertTransaction(ctx, db.Pool(), changed)
	if err != nil {
		t.Fatalf("second insert: %v", err)
	}
	if result != Conflict {
		t.Fatalf("result = %s, want CONFLICT", result)
	}

	existing, err := fetchTransaction(ctx, db.Pool(), txid)
	if err != nil {
		t.Fatalf("fetchTransaction: %v", err)
	}
	if existing.Fee != 105655 {
		t.Fatalf("existing row was overwritten: Fee = %d, want unchanged 105655", existing.Fee)
	}
}