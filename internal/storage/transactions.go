// internal/storage/transactions.go
package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dipak0000812/TraceLayer/internal/domain"
)

const uniqueViolationCode = "23505"

// InsertResult reports what InsertTransaction actually did, per
// DATA_CONTRACT.md section 3's duplicate policy: identical payload for an
// existing txid is a harmless no-op; a different payload for an existing
// txid is a conflict. Neither case ever overwrites the stored row.
type InsertResult int

const (
	Inserted InsertResult = iota
	DuplicateNoop
	Conflict
)

func (r InsertResult) String() string {
	switch r {
	case Inserted:
		return "INSERTED"
	case DuplicateNoop:
		return "DUPLICATE_NOOP"
	case Conflict:
		return "CONFLICT"
	default:
		return "UNKNOWN"
	}
}

func InsertTransaction(ctx context.Context, pool *pgxpool.Pool, tx domain.Transaction) (InsertResult, error) {
	inputAddrs, err := json.Marshal(tx.InputAddresses)
	if err != nil {
		return 0, fmt.Errorf("storage: marshaling input_addresses: %w", err)
	}
	outputAddrs, err := json.Marshal(tx.OutputAddresses)
	if err != nil {
		return 0, fmt.Errorf("storage: marshaling output_addresses: %w", err)
	}
	inputAmounts, err := marshalAmounts(tx.InputAmounts)
	if err != nil {
		return 0, fmt.Errorf("storage: marshaling input_amounts: %w", err)
	}
	outputAmounts, err := marshalAmounts(tx.OutputAmounts)
	if err != nil {
		return 0, fmt.Errorf("storage: marshaling output_amounts: %w", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO transactions (
			txid, block_time, input_addresses, output_addresses,
			input_amounts, output_amounts, fee, script_type,
			provenance, dataset_id, generator_version
		) VALUES ($1, $2, $3::jsonb, $4::jsonb, $5::jsonb, $6::jsonb, $7::numeric, $8, $9, $10, $11)`,
		tx.TXID, tx.BlockTime, string(inputAddrs), string(outputAddrs),
		string(inputAmounts), string(outputAmounts), tx.Fee.String(),
		string(tx.ScriptType), string(tx.Provenance), tx.DatasetID, tx.GeneratorVersion,
	)
	if err == nil {
		return Inserted, nil
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != uniqueViolationCode {
		return 0, fmt.Errorf("storage: inserting transaction %s: %w", tx.TXID, err)
	}

	existing, fetchErr := fetchTransaction(ctx, pool, tx.TXID)
	if fetchErr != nil {
		return 0, fmt.Errorf("storage: txid %s violated uniqueness but re-fetch failed: %w", tx.TXID, fetchErr)
	}
	if transactionsEqual(existing, tx) {
		return DuplicateNoop, nil
	}
	return Conflict, nil
}

func fetchTransaction(ctx context.Context, pool *pgxpool.Pool, txid string) (domain.Transaction, error) {
	row := pool.QueryRow(ctx, `
		SELECT txid, block_time, input_addresses::text, output_addresses::text,
		       input_amounts::text, output_amounts::text, fee::text,
		       script_type, provenance, dataset_id, generator_version
		FROM transactions WHERE txid = $1`, txid)

	var tx domain.Transaction
	var inputAddrsJSON, outputAddrsJSON, inputAmountsJSON, outputAmountsJSON, feeText, scriptType, provenance string

	if err := row.Scan(&tx.TXID, &tx.BlockTime, &inputAddrsJSON, &outputAddrsJSON,
		&inputAmountsJSON, &outputAmountsJSON, &feeText, &scriptType, &provenance,
		&tx.DatasetID, &tx.GeneratorVersion); err != nil {
		return domain.Transaction{}, fmt.Errorf("storage: scanning transaction row: %w", err)
	}
	if err := json.Unmarshal([]byte(inputAddrsJSON), &tx.InputAddresses); err != nil {
		return domain.Transaction{}, fmt.Errorf("storage: decoding input_addresses: %w", err)
	}
	if err := json.Unmarshal([]byte(outputAddrsJSON), &tx.OutputAddresses); err != nil {
		return domain.Transaction{}, fmt.Errorf("storage: decoding output_addresses: %w", err)
	}
	inputAmounts, err := unmarshalAmounts(inputAmountsJSON)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("storage: decoding input_amounts: %w", err)
	}
	outputAmounts, err := unmarshalAmounts(outputAmountsJSON)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("storage: decoding output_amounts: %w", err)
	}
	fee, err := domain.ParseBTCString(feeText)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("storage: decoding fee: %w", err)
	}

	tx.InputAmounts, tx.OutputAmounts, tx.Fee = inputAmounts, outputAmounts, fee
	tx.ScriptType, tx.Provenance = domain.ScriptType(scriptType), domain.Provenance(provenance)
	return tx, nil
}

func marshalAmounts(amounts []domain.Satoshis) ([]byte, error) {
	strs := make([]string, len(amounts))
	for i, a := range amounts {
		strs[i] = a.String()
	}
	return json.Marshal(strs)
}

func unmarshalAmounts(raw string) ([]domain.Satoshis, error) {
	var strs []string
	if err := json.Unmarshal([]byte(raw), &strs); err != nil {
		return nil, err
	}
	out := make([]domain.Satoshis, len(strs))
	for i, s := range strs {
		v, err := domain.ParseBTCString(s)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

// transactionsEqual avoids reflect.DeepEqual because time.Time's internal
// monotonic/location representation can differ between an in-memory value
// and one round-tripped through Postgres even when both are the same instant.
func transactionsEqual(a, b domain.Transaction) bool {
	if a.TXID != b.TXID || !a.BlockTime.Equal(b.BlockTime) || a.Fee != b.Fee ||
		a.ScriptType != b.ScriptType || a.Provenance != b.Provenance ||
		a.DatasetID != b.DatasetID || a.GeneratorVersion != b.GeneratorVersion {
		return false
	}
	return stringSlicesEqual(a.InputAddresses, b.InputAddresses) &&
		stringSlicesEqual(a.OutputAddresses, b.OutputAddresses) &&
		amountSlicesEqual(a.InputAmounts, b.InputAmounts) &&
		amountSlicesEqual(a.OutputAmounts, b.OutputAmounts)
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func amountSlicesEqual(a, b []domain.Satoshis) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

var ErrTransactionNotFound = errors.New("storage: transaction not found")

// GetTransaction fetches one transaction by txid for read-side consumers
// (evidence assembly, lead detail endpoints). Distinguishes "not found"
// from real failures so callers can map it to a 404 later in Phase 11.
func GetTransaction(ctx context.Context, pool *pgxpool.Pool, txid string) (domain.Transaction, error) {
	tx, err := fetchTransaction(ctx, pool, txid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Transaction{}, fmt.Errorf("%w: %s", ErrTransactionNotFound, txid)
		}
		return domain.Transaction{}, err
	}
	return tx, nil
}
