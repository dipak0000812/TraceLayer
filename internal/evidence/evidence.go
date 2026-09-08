// internal/evidence/evidence.go
package evidence

import (
	"context"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dipak0000812/TraceLayer/internal/domain"
	"github.com/dipak0000812/TraceLayer/internal/storage"
)

// Evidence is the joined chain+network view for one transaction, consumed
// by fusion orchestration (Phase 9) and lead assembly (Phase 10).
// NetworkQuality is Q(tx) per the frozen methodology — a measure of
// evidence reliability, NOT a probability of transaction origin.
type Evidence struct {
	Transaction    domain.Transaction
	Observations   []domain.NetworkObservation
	NetworkQuality float64
}

// GetEvidence assembles the evidence view for one txid. Correlation must
// have already run (Phase 5) for recently-ingested observations to appear
// here — this function reads persisted correlation_status, it does not
// re-derive it.
func GetEvidence(ctx context.Context, pool *pgxpool.Pool, txid string) (Evidence, error) {
	tx, err := storage.GetTransaction(ctx, pool, txid)
	if err != nil {
		return Evidence{}, fmt.Errorf("evidence: %w", err)
	}

	observations, err := storage.GetCorrelatedObservations(ctx, pool, txid)
	if err != nil {
		return Evidence{}, fmt.Errorf("evidence: %w", err)
	}

	return Evidence{
		Transaction:    tx,
		Observations:   observations,
		NetworkQuality: computeNetworkQuality(observations),
	}, nil
}

// computeNetworkQuality implements the frozen Q(tx) methodology:
//
//	Q(tx) = 0                                           when N == 0
//	Q(tx) = min(1, N/3) * max(0, 1 - timing_spread/120) when N > 0
//
// N is the observation count, timing_spread is the number of seconds
// between the earliest and latest observed_at among this txid's correlated
// observations (N_threshold=3, MAX_SPREAD=120s). This is a methodology
// metric, not proof of transaction origin.
func computeNetworkQuality(observations []domain.NetworkObservation) float64 {
	n := len(observations)
	if n == 0 {
		return 0
	}

	minT, maxT := observations[0].ObservedAt, observations[0].ObservedAt
	for _, o := range observations[1:] {
		if o.ObservedAt.Before(minT) {
			minT = o.ObservedAt
		}
		if o.ObservedAt.After(maxT) {
			maxT = o.ObservedAt
		}
	}
	timingSpread := maxT.Sub(minT).Seconds()

	countFactor := math.Min(1, float64(n)/3)
	spreadFactor := math.Max(0, 1-timingSpread/120)
	return countFactor * spreadFactor
}
