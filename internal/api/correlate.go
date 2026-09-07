package api

import (
	"net/http"
	"os"
	"time"

	"github.com/dipak0000812/TraceLayer/internal/correlation"
	"github.com/dipak0000812/TraceLayer/internal/domain"
	"github.com/dipak0000812/TraceLayer/internal/entity"
	"github.com/dipak0000812/TraceLayer/internal/intelligence"
	"github.com/dipak0000812/TraceLayer/internal/ranking"
)

type correlateResponse struct {
	Status                 string `json:"status"`
	EntitiesClustered      int    `json:"entities_clustered"`
	LeadsGenerated         int    `json:"leads_generated"`
	ObservationsCorrelated int    `json:"observations_correlated"`
}

// correlateHandler backs POST /api/v1/correlate.
func correlateHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		corrResult, err := correlation.Run(r.Context(), deps.Pool)
		if err != nil {
			writeError(w, http.StatusInternalServerError, CodeInternalError, "correlation: "+err.Error(), nil)
			return
		}
		entityResult, err := entity.Run(r.Context(), deps.Pool, domain.ProvenanceDerived, "6bc084b677a63411")
		if err != nil {
			writeError(w, http.StatusInternalServerError, CodeInternalError, "entity clustering: "+err.Error(), nil)
			return
		}
		intelURL := os.Getenv("INTELLIGENCE_URL")
		var intelClient *intelligence.Client
		if intelURL != "" {
			intelClient = intelligence.NewClient(intelURL, 5*time.Second)
		}
		leadsCount, err := ranking.Run(r.Context(), deps.Pool, intelClient)
		if err != nil {
			writeError(w, http.StatusInternalServerError, CodeInternalError, "ranking: "+err.Error(), nil)
			return
		}
		writeJSON(w, http.StatusOK, correlateResponse{
			Status:                 "SUCCESS",
			EntitiesClustered:      entityResult.EntitiesClustered,
			LeadsGenerated:         leadsCount,
			ObservationsCorrelated: corrResult.Correlated,
		})
	}
}
