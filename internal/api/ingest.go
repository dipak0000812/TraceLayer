package api

import (
	"net/http"

	"github.com/dipak0000812/TraceLayer/internal/ingestion"
	"github.com/dipak0000812/TraceLayer/internal/storage"
)

type batchSummary struct {
	IngestedCount  int    `json:"ingested_count"`
	DuplicateCount int    `json:"duplicate_count"`
	RejectedCount  int    `json:"rejected_count"`
	DatasetID      string `json:"dataset_id"`
}

// ingestBlockchainHandler backs POST /api/v1/ingest/blockchain. CSV bodies
// only — see decision 5 above re: JSON-array support gap.
func ingestBlockchainHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		valid, rejections, err := ingestion.ParseTransactionsCSV(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, CodeValidationFailed, err.Error(), nil)
			return
		}

		summary := batchSummary{RejectedCount: len(rejections)}
		for _, tx := range valid {
			result, err := storage.InsertTransaction(r.Context(), deps.Pool, tx)
			if err != nil {
				writeError(w, http.StatusInternalServerError, CodeInternalError, "storing transaction: "+err.Error(), nil)
				return
			}
			switch result {
			case storage.Inserted:
				summary.IngestedCount++
				summary.DatasetID = tx.DatasetID
			case storage.DuplicateNoop:
				summary.DuplicateCount++
			case storage.Conflict:
				summary.RejectedCount++ // same txid, different payload — rejected, never overwritten
			}
		}
		writeJSON(w, http.StatusOK, summary)
	}
}

func ingestNetworkHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		valid, rejections, err := ingestion.ParseNetworkObservationsCSV(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, CodeValidationFailed, err.Error(), nil)
			return
		}

		summary := batchSummary{RejectedCount: len(rejections)}
		for _, obs := range valid {
			result, err := storage.InsertNetworkObservation(r.Context(), deps.Pool, obs)
			if err != nil {
				writeError(w, http.StatusInternalServerError, CodeInternalError, "storing observation: "+err.Error(), nil)
				return
			}
			switch result {
			case storage.NetworkInserted:
				summary.IngestedCount++
				summary.DatasetID = obs.DatasetID
			case storage.NetworkDuplicate:
				summary.DuplicateCount++
			}
		}
		writeJSON(w, http.StatusOK, summary)
	}
}
