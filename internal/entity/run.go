// internal/entity/run.go
package entity

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dipak0000812/TraceLayer/internal/domain"
	"github.com/dipak0000812/TraceLayer/internal/storage"
)

// RunResult reports what one clustering pass did — this is the number
// POST /api/v1/correlate's entities_clustered field should stop hardcoding
// to 0 once you wire this in.
type RunResult struct {
	EntitiesClustered int
}

// Run reads every transaction's input_addresses, clusters them, and
// persists each resulting entity. provenance/datasetID are supplied by the
// caller rather than derived per-address — fine for Round-2's single
// seed-42 dataset; would need revisiting if multiple datasets are ever
// ingested into the same table.
func Run(ctx context.Context, pool *pgxpool.Pool, provenance domain.Provenance, datasetID string) (RunResult, error) {
	groups, err := storage.GetAllInputAddressGroups(ctx, pool)
	if err != nil {
		return RunResult{}, fmt.Errorf("entity: %w", err)
	}

	entities := Cluster(groups)

	for _, e := range entities {
		if err := e.Validate(); err != nil {
			return RunResult{}, fmt.Errorf("entity: clustered entity failed validation: %w", err)
		}
		if err := storage.InsertEntity(ctx, pool, e, provenance, datasetID); err != nil {
			return RunResult{}, fmt.Errorf("entity: %w", err)
		}
	}

	return RunResult{EntitiesClustered: len(entities)}, nil
}