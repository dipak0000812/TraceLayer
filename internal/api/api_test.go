package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dipak0000812/TraceLayer/internal/storage"
)

func testDeps(t *testing.T) Deps {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping live API test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	db, err := storage.New(ctx, storage.Config{DSN: dsn, ConnectTimeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("storage.New: %v", err)
	}
	t.Cleanup(db.Close)
	return Deps{DB: db, Pool: db.Pool()}
}

func TestHealthHandler_Live(t *testing.T) {
	deps := testDeps(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	healthHandler(deps)(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"postgres":"UP"`) {
		t.Fatalf("expected postgres UP in body, got %s", rec.Body.String())
	}
}

func TestEvidenceHandler_UnknownTXID_404(t *testing.T) {
	deps := testDeps(t)
	txid := strings.Repeat("0", 60) + "dead"
	if len(txid) != 64 {
		t.Fatalf("test fixture bug: txid is %d chars, want 64", len(txid))
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/"+txid, nil)
	req.SetPathValue("txid", txid)
	rec := httptest.NewRecorder()
	evidenceHandler(deps)(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", rec.Code, rec.Body.String())
	}
}

func TestEvidenceHandler_InvalidTXID_400(t *testing.T) {
	deps := testDeps(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/not-a-txid", nil)
	req.SetPathValue("txid", "not-a-txid")
	rec := httptest.NewRecorder()
	evidenceHandler(deps)(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}
