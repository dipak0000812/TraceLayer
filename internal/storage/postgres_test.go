// internal/storage/postgres_test.go
package storage

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestConfigFromEnv_MissingDATABASE_URL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := ConfigFromEnv(); err == nil {
		t.Fatal("expected error when DATABASE_URL is unset, got nil")
	}
}

func TestNew_RequiresDSN(t *testing.T) {
	if _, err := New(context.Background(), Config{}); err == nil {
		t.Fatal("expected error for empty DSN, got nil")
	}
}

func TestNew_MalformedDSN(t *testing.T) {
	_, err := New(context.Background(), Config{DSN: "not-a-valid-dsn", ConnectTimeout: time.Second})
	if err == nil {
		t.Fatal("expected error for malformed DSN, got nil")
	}
}

func TestNew_UnreachableHost_FailsFastNotHang(t *testing.T) {
	start := time.Now()
	_, err := New(context.Background(), Config{
		DSN:            "postgres://user:pass@192.0.2.1:5432/db?sslmode=disable", // RFC 5737 TEST-NET-1, never routable
		ConnectTimeout: 2 * time.Second,
	})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected error connecting to an unreachable host, got nil")
	}
	if elapsed > 10*time.Second {
		t.Fatalf("New() took %s against an unreachable host — ConnectTimeout is not bounding it", elapsed)
	}
}

func TestNew_LiveDatabase(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping live Postgres test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := New(ctx, Config{DSN: dsn, ConnectTimeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("New() with live DATABASE_URL failed: %v", err)
	}
	defer db.Close()

	if err := db.HealthCheck(ctx); err != nil {
		t.Fatalf("HealthCheck() failed against live DB: %v", err)
	}

	db.Close() // must be safe to call twice
	db.Close()
}