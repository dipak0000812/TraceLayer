package db

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/dipak0000812/TraceLayer/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository provides an interface to the PostgreSQL database
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository initializes a new connection pool
func NewPostgresRepository(ctx context.Context, dsn string) (*PostgresRepository, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	log.Println("Successfully connected to PostgreSQL")
	return &PostgresRepository{pool: pool}, nil
}

// Close closes the connection pool
func (r *PostgresRepository) Close() {
	r.pool.Close()
}

// UpsertTransaction inserts a transaction or ignores it if the txid already exists (idempotent)
func (r *PostgresRepository) UpsertTransaction(ctx context.Context, tx domain.Transaction) error {
	inputAddrs, _ := json.Marshal(tx.InputAddresses)
	outputAddrs, _ := json.Marshal(tx.OutputAddresses)
	inputAmts, _ := json.Marshal(tx.InputAmounts)
	outputAmts, _ := json.Marshal(tx.OutputAmounts)

	query := `
		INSERT INTO transactions (
			txid, block_time, input_addresses, output_addresses, 
			input_amounts, output_amounts, fee, script_type, 
			provenance, dataset_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (txid) DO NOTHING
	`

	_, err := r.pool.Exec(ctx, query,
		tx.TxID,
		tx.BlockTime,
		inputAddrs,
		outputAddrs,
		inputAmts,
		outputAmts,
		tx.Fee,
		tx.ScriptType,
		tx.Provenance,
		tx.DatasetID,
	)
	return err
}

// InsertNetworkObservation inserts a new network observation
func (r *PostgresRepository) InsertNetworkObservation(ctx context.Context, obs domain.NetworkObservation) error {
	query := `
		INSERT INTO network_observations (
			observation_id, observed_txid, observed_at, src_ip, dst_ip, 
			src_port, dst_port, geo_country, asn, provenance, dataset_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (observation_id) DO NOTHING
	`

	_, err := r.pool.Exec(ctx, query,
		obs.ObservationID,
		obs.ObservedTxID,
		obs.ObservedAt,
		obs.SrcIP,
		obs.DstIP,
		obs.SrcPort,
		obs.DstPort,
		obs.GeoCountry,
		obs.ASN,
		obs.Provenance,
		obs.DatasetID,
	)
	return err
}

// IngestionStats holds metrics about the database ingestion state
type IngestionStats struct {
	TotalTransactions int
	TotalObservations int
	Correlated        int
	Orphans           int
}

// GetIngestionStatus returns counts of the current records
func (r *PostgresRepository) GetIngestionStatus(ctx context.Context) (*IngestionStats, error) {
	stats := &IngestionStats{}

	// Count transactions
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM transactions").Scan(&stats.TotalTransactions)
	if err != nil {
		return nil, err
	}

	// Count observations
	err = r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM network_observations").Scan(&stats.TotalObservations)
	if err != nil {
		return nil, err
	}

	// Count correlated vs orphans (observations where the txid exists in transactions table)
	queryCorrelated := `
		SELECT 
			SUM(CASE WHEN t.txid IS NOT NULL THEN 1 ELSE 0 END) as correlated,
			SUM(CASE WHEN t.txid IS NULL THEN 1 ELSE 0 END) as orphans
		FROM network_observations n
		LEFT JOIN transactions t ON n.observed_txid = t.txid
	`
	
	// Handle potential NULLs if table is empty
	var correlated, orphans *int
	err = r.pool.QueryRow(ctx, queryCorrelated).Scan(&correlated, &orphans)
	if err != nil {
		return nil, err
	}

	if correlated != nil {
		stats.Correlated = *correlated
	}
	if orphans != nil {
		stats.Orphans = *orphans
	}

	return stats, nil
}
