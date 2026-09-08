// internal/storage/entity.go
package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dipak0000812/TraceLayer/internal/domain"
)

// GetAllInputAddressGroups returns every transaction's input_addresses, one
// []string per transaction, for entity.Cluster to consume.
func GetAllInputAddressGroups(ctx context.Context, pool *pgxpool.Pool) ([][]string, error) {
	rows, err := pool.Query(ctx, `SELECT input_addresses::text FROM transactions`)
	if err != nil {
		return nil, fmt.Errorf("storage: querying input_addresses: %w", err)
	}
	defer rows.Close()

	var groups [][]string
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("storage: scanning input_addresses: %w", err)
		}
		var addrs []string
		if err := json.Unmarshal([]byte(raw), &addrs); err != nil {
			return nil, fmt.Errorf("storage: decoding input_addresses: %w", err)
		}
		groups = append(groups, addrs)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: iterating transactions: %w", err)
	}
	return groups, nil
}

// InsertEntity persists one entity. entity_id is deterministic (see
// entity.Cluster), so ON CONFLICT DO NOTHING makes re-running clustering on
// unchanged data a safe no-op — same idempotency pattern as
// InsertNetworkObservation. See Phase 8's flagged limitation on stale rows
// when clustering changes across re-runs; not handled here, deliberately.
func InsertEntity(ctx context.Context, pool *pgxpool.Pool, e domain.Entity, provenance domain.Provenance, datasetID string) error {
	members, err := json.Marshal(e.MemberAddresses)
	if err != nil {
		return fmt.Errorf("storage: marshaling member_addresses: %w", err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO entities (entity_id, member_addresses, cluster_size, provenance, dataset_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (entity_id) DO NOTHING`,
		e.EntityID, string(members), e.ClusterSize, string(provenance), datasetID,
	)
	if err != nil {
		return fmt.Errorf("storage: inserting entity %s: %w", e.EntityID, err)
	}
	return nil
}