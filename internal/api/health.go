package api

import (
	"net/http"
	"time"
)

type healthResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

// healthHandler backs GET /health. Neo4j and the intelligence worker aren't
// wired into the Go process yet — reported as NOT_IMPLEMENTED rather than a
// fabricated "UP".
func healthHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		services := map[string]string{
			"neo4j":               "NOT_IMPLEMENTED",
			"intelligence_worker": "NOT_IMPLEMENTED",
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
