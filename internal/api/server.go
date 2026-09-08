package api

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dipak0000812/TraceLayer/internal/storage"
)

// Deps is every dependency a handler needs, passed explicitly — no
// package-level globals.
type Deps struct {
	DB   *storage.DB
	Pool *pgxpool.Pool
}

// NewServer wires Phase 7's routes. Uses Go 1.22's method-pattern ServeMux
// (go.mod already requires 1.22.2) instead of a router dependency —
// nothing here needs more than that.
func NewServer(addr string, deps Deps) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler(deps))
	mux.HandleFunc("POST /api/v1/ingest/blockchain", ingestBlockchainHandler(deps))
	mux.HandleFunc("POST /api/v1/ingest/network", ingestNetworkHandler(deps))
	mux.HandleFunc("POST /api/v1/correlate", correlateHandler(deps))
	mux.HandleFunc("GET /api/v1/evidence/{txid}", evidenceHandler(deps))
	mux.HandleFunc("GET /api/v1/leads", leadsListHandler(deps))
	mux.HandleFunc("GET /api/v1/leads/{id}", leadDetailHandler(deps))

	return &http.Server{
		Addr:              addr,
		Handler:           withCORS(withRequestTimeout(mux, 30*time.Second)),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      35 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func withRequestTimeout(next http.Handler, d time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), d)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
