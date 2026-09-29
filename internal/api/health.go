package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"time"
)

type healthResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

// workerHealthState probes the Python intelligence worker's /health endpoint
// and returns a concise status string. It never panics and always returns
// within the supplied timeout.
//
// States returned:
//   - "UP"               – worker reachable and model_loaded = true
//   - "MODEL_NOT_LOADED" – worker reachable but model_loaded = false
//   - "UNREACHABLE"      – network error or non-2xx response
//   - "NOT_CONFIGURED"   – INTELLIGENCE_URL is empty
func workerHealthState(timeout time.Duration) string {
	url := os.Getenv("INTELLIGENCE_URL")
	if url == "" {
		return "NOT_CONFIGURED"
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url+"/health", nil)
	if err != nil {
		return "UNREACHABLE"
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "UNREACHABLE"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "UNREACHABLE"
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "UNREACHABLE"
	}

	var payload struct {
		ModelLoaded bool `json:"model_loaded"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "UNREACHABLE"
	}
	if payload.ModelLoaded {
		return "UP"
	}
	return "MODEL_NOT_LOADED"
}

// healthHandler backs GET /health. It checks PostgreSQL connectivity and
// probes the intelligence worker. The neo4j key is not present — Neo4j is
// not part of the Round-2 architecture.
func healthHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Probe intelligence worker with a short timeout so health checks
		// stay fast even when the worker is slow.
		intelState := workerHealthState(3 * time.Second)

		services := map[string]string{
			"intelligence_worker": intelState,
		}
		status, httpStatus := "HEALTHY", http.StatusOK

		if err := deps.DB.HealthCheck(r.Context()); err != nil {
			services["postgres"] = "DOWN"
			status, httpStatus = "DEGRADED", http.StatusServiceUnavailable
		} else {
			services["postgres"] = "UP"
		}

		writeJSON(w, httpStatus, healthResponse{
			Status:    status,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Services:  services,
		})
	}
}

