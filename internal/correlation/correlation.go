// internal/correlation/correlation.go
package correlation

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RunResult reports what one correlation pass actually changed.
type RunResult struct {
	Correlated int
	Orphaned   int
	Duration   time.Duration
}

// Run re-evaluates every network_observations row currently in PENDING or
// ORPHAN state against the transactions table and transitions each to
// CORRELATED or ORPHAN. ORPHAN is retryable by design — per ARCHITECTURE.md,
// network observations may arrive before their transaction, so a row marked
// ORPHAN today may correlate on a later run once the matching txid is
// ingested. CORRELATED is terminal: once a row leaves PENDING/ORPHAN it's
// excluded from the WHERE clause and this function never touches it again,
// so re-running is always safe — a pass with nothing new to correlate
// changes zero rows.
func Run(ctx context.Context, pool *pgxpool.Pool) (RunResult, error) {
	start := time.Now()

	rows, err := pool.Query(ctx, `
		WITH updated AS (
			UPDATE network_observations
			SET correlation_status = CASE
				WHEN EXISTS (
					SELECT 1 FROM transactions t WHERE t.txid = network_observations.observed_txid
				) THEN 'CORRELATED' ELSE 'ORPHAN' END
			WHERE correlation_status IN ('PENDING', 'ORPHAN')
			RETURNING correlation_status
		)
		SELECT correlation_status, count(*) FROM updated GROUP BY correlation_status
	`)
	if err != nil {
		return RunResult{}, fmt.Errorf("correlation: running pass: %w", err)
	}
	defer rows.Close()

	var result RunResult
	for rows.Next() {
		var status string
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			return RunResult{}, fmt.Errorf("correlation: scanning result: %w", err)
		}
		switch status {
		case "CORRELATED":
			result.Correlated = n
		case "ORPHAN":
			result.Orphaned = n
		}
	}
	if err := rows.Err(); err != nil {
		return RunResult{}, fmt.Errorf("correlation: iterating results: %w", err)
	}

	result.Duration = time.Since(start)
	return result, nil
}