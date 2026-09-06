// internal/storage/postgres.go
package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config controls pgxpool construction. Every field is explicit — there is
// no hardcoded default connection string anywhere in this package.
type Config struct {
	DSN string // e.g. "postgres://user:pass@host:5432/dbname?sslmode=disable"

	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	ConnectTimeout  time.Duration
}

// ConfigFromEnv reads DATABASE_URL. Its absence is a fail-fast error, not a
// fallback to a local/dev connection string.
func ConfigFromEnv() (Config, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return Config{}, errors.New("storage: DATABASE_URL is not set")
	}
	return Config{
		DSN:             dsn,
		MaxConns:        10,
		MinConns:        1,
		MaxConnLifetime: 30 * time.Minute,
		MaxConnIdleTime: 5 * time.Minute,
		ConnectTimeout:  5 * time.Second,
	}, nil
}

// DB wraps a pgxpool.Pool. It exposes only lifecycle and health — table
// queries belong to the packages that own that data (Phase 3+), never here.
type DB struct {
	pool      *pgxpool.Pool
	closeOnce sync.Once
}

// New builds a pool from cfg and verifies connectivity with one bounded
// Ping before returning. Both pool construction and the ping are bounded by
// cfg.ConnectTimeout — this never blocks indefinitely against an
// unreachable host.
func New(ctx context.Context, cfg Config) (*DB, error) {
	if cfg.DSN == "" {
		return nil, errors.New("storage: Config.DSN must not be empty")
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("storage: parsing DSN: %w", err)
	}
	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = cfg.MaxConns
	}
	if cfg.MinConns > 0 {
		poolCfg.MinConns = cfg.MinConns
	}
	if cfg.MaxConnLifetime > 0 {
		poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	}
	if cfg.MaxConnIdleTime > 0 {
		poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	}

	connectTimeout := cfg.ConnectTimeout
	if connectTimeout <= 0 {
		connectTimeout = 5 * time.Second
	}

	ctorCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctorCtx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("storage: constructing pool: %w", err)
	}

	pingCtx, cancel2 := context.WithTimeout(ctx, connectTimeout)
	defer cancel2()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("storage: initial ping failed: %w", err)
	}

	return &DB{pool: pool}, nil
}

// Pool exposes the underlying pool for packages that own their own queries.
// This package does not wrap or hide query methods — only lifecycle.
func (db *DB) Pool() *pgxpool.Pool { return db.pool }

// HealthCheck backs GET /health's "postgres" status (Phase 7).
func (db *DB) HealthCheck(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

// Close shuts down the pool. Safe to call any number of times or
// concurrently — only the first call has any effect.
func (db *DB) Close() {
	db.closeOnce.Do(func() {
		db.pool.Close()
	})
}