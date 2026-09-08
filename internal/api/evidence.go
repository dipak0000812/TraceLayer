package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/dipak0000812/TraceLayer/internal/domain"
	"github.com/dipak0000812/TraceLayer/internal/evidence"
	"github.com/dipak0000812/TraceLayer/internal/storage"
)

type evidenceResponse struct {
	Transaction         evidenceTransaction   `json:"transaction"`
	NetworkObservations []evidenceObservation `json:"network_observations"`
	QualityMetric       float64               `json:"quality_metric"`
	Disclaimer          string                `json:"disclaimer"`
}

type evidenceTransaction struct {
	TXID            string          `json:"txid"`
	Timestamp       string          `json:"timestamp"`
	InputAddresses  []string        `json:"input_addresses"`
	OutputAddresses []string        `json:"output_addresses"`
	Fee             domain.Satoshis `json:"fee"`
	ScriptType      string          `json:"script_type"`
}

type evidenceObservation struct {
	ObservationID      string  `json:"observation_id"`
	ObservedAt         string  `json:"observed_at"`
	FirstHeardPeerIP   string  `json:"first_heard_peer_ip"`
	SrcPort            uint16  `json:"src_port"`
	GeoCountry         *string `json:"geo_country"`
	ASN                *string `json:"asn"`
	PropagationDelayMS *int64  `json:"propagation_delay_ms"`
}

const evidenceDisclaimer = "First-heard peer IP indicates vantage relay observation, NOT cryptographic sender identity."

func evidenceHandler(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		txid := r.PathValue("txid")
		if !domain.ValidTXID(txid) {
			field := "txid"
			writeError(w, http.StatusBadRequest, CodeValidationFailed, "txid must be 64 lowercase hex characters", &field)
			return
		}

		ev, err := evidence.GetEvidence(r.Context(), deps.Pool, txid)
		if err != nil {
			if errors.Is(err, storage.ErrTransactionNotFound) {
				writeError(w, http.StatusNotFound, CodeNotFound, "transaction not found", nil)
				return
			}
			writeError(w, http.StatusInternalServerError, CodeInternalError, "assembling evidence: "+err.Error(), nil)
			return
		}

		obs := make([]evidenceObservation, len(ev.Observations))
		for i, o := range ev.Observations {
			obs[i] = evidenceObservation{
				ObservationID:      o.ObservationID,
				ObservedAt:         o.ObservedAt.UTC().Format(time.RFC3339),
				FirstHeardPeerIP:   o.SrcIP.String(),
				SrcPort:            o.SrcPort,
				GeoCountry:         o.GeoCountry,
				ASN:                o.ASN,
				PropagationDelayMS: o.PropagationDelayMS,
			}
		}

		writeJSON(w, http.StatusOK, evidenceResponse{
			Transaction: evidenceTransaction{
				TXID:            ev.Transaction.TXID,
				Timestamp:       ev.Transaction.BlockTime.UTC().Format(time.RFC3339),
				InputAddresses:  ev.Transaction.InputAddresses,
				OutputAddresses: ev.Transaction.OutputAddresses,
				Fee:             ev.Transaction.Fee,
				ScriptType:      string(ev.Transaction.ScriptType),
			},
			NetworkObservations: obs,
			QualityMetric:       ev.NetworkQuality,
			Disclaimer:          evidenceDisclaimer,
		})
	}
}
