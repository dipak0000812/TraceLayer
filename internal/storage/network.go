// internal/storage/network.go
package storage

import (
	"context"
	"errors"
	"fmt"
	"net/netip"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dipak0000812/TraceLayer/internal/domain"
)

// NetworkInsertResult reports what InsertNetworkObservation did. Unlike
// transactions, network observations have no payload-conflict case per
// DATA_CONTRACT.md §3 — a repeated observation_id is always a duplicate,
// never compared field-by-field. Either way the stored row is never touched.
type NetworkInsertResult int

const (
	NetworkInserted NetworkInsertResult = iota
	NetworkDuplicate
)

func (r NetworkInsertResult) String() string {
	switch r {
	case NetworkInserted:
		return "INSERTED"
	case NetworkDuplicate:
		return "DUPLICATE"
	default:
		return "UNKNOWN"
	}
}

// InsertNetworkObservation reuses uniqueViolationCode defined in
// transactions.go (same package) — no redeclaration needed.
func InsertNetworkObservation(ctx context.Context, pool *pgxpool.Pool, obs domain.NetworkObservation) (NetworkInsertResult, error) {
	_, err := pool.Exec(ctx, `
    INSERT INTO network_observations (
        observation_id, observed_txid, observed_at, src_ip, dst_ip,
        src_port, dst_port, correlation_status, provenance, dataset_id, generator_version
    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		obs.ObservationID, obs.ObservedTXID, obs.ObservedAt,
		obs.SrcIP.String(), obs.DstIP.String(),
		int32(obs.SrcPort), int32(obs.DstPort),
		string(domain.CorrelationPending), string(obs.Provenance),
		obs.DatasetID, obs.GeneratorVersion,
	)
	if err == nil {
		return NetworkInserted, nil
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != uniqueViolationCode {
		return 0, fmt.Errorf("storage: inserting network observation %s: %w", obs.ObservationID, err)
	}
	return NetworkDuplicate, nil
}

// GetCorrelatedObservations returns every CORRELATED observation for a
// txid, ordered by observed_at. Deliberately excludes PENDING/ORPHAN — see
// the engineering decision in Phase 6: only persisted correlation matches
// count as evidence, this function doesn't re-derive anything.
func GetCorrelatedObservations(ctx context.Context, pool *pgxpool.Pool, txid string) ([]domain.NetworkObservation, error) {
	rows, err := pool.Query(ctx, `
		SELECT observation_id, observed_txid, observed_at, src_ip, dst_ip,
		       src_port, dst_port, correlation_status, provenance, dataset_id, generator_version
		FROM network_observations
		WHERE observed_txid = $1 AND correlation_status = 'CORRELATED'
		ORDER BY observed_at`, txid)
	if err != nil {
		return nil, fmt.Errorf("storage: querying correlated observations for %s: %w", txid, err)
	}
	defer rows.Close()

	var out []domain.NetworkObservation
	for rows.Next() {
		var o domain.NetworkObservation
		var srcIPStr, dstIPStr, status, provenance string
		var srcPort, dstPort int32

		if err := rows.Scan(&o.ObservationID, &o.ObservedTXID, &o.ObservedAt, &srcIPStr, &dstIPStr,
			&srcPort, &dstPort, &status, &provenance, &o.DatasetID, &o.GeneratorVersion); err != nil {
			return nil, fmt.Errorf("storage: scanning observation row: %w", err)
		}
		srcIP, err := netip.ParseAddr(srcIPStr)
		if err != nil {
			return nil, fmt.Errorf("storage: decoding src_ip: %w", err)
		}
		dstIP, err := netip.ParseAddr(dstIPStr)
		if err != nil {
			return nil, fmt.Errorf("storage: decoding dst_ip: %w", err)
		}
		o.SrcIP, o.DstIP = srcIP, dstIP
		o.SrcPort, o.DstPort = uint16(srcPort), uint16(dstPort)
		o.CorrelationStatus = domain.CorrelationStatus(status)
		o.Provenance = domain.Provenance(provenance)
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: iterating observations: %w", err)
	}
	return out, nil
}
