// internal/intelligence/features.go
package intelligence

import (
	"strconv"
	"time"

	"github.com/dipak0000812/TraceLayer/internal/domain"
	"github.com/dipak0000812/TraceLayer/internal/evidence"
)

// BuildFeatures converts an assembled Evidence bundle (Phase 6) into the
// feature shape POST /intelligence/score expects.
//
// amount_btc is sum(output_amounts) — an ASSUMPTION, not a contract fact;
// the contract's single-output example doesn't disambiguate multi-output
// transactions. Flag if a different definition is needed.
//
// amount_btc/fee_btc are float64 here deliberately: this is a transient ML
// feature vector for a model that already computes in floating point, not
// persisted/displayed ledger data — the precision invariant behind
// Satoshis.MarshalJSON doesn't apply in this direction.
func BuildFeatures(ev evidence.Evidence) TransactionFeatures {
	var total domain.Satoshis
	for _, a := range ev.Transaction.OutputAmounts {
		total += a
	}
	amountBTC, _ := strconv.ParseFloat(total.String(), 64)
	feeBTC, _ := strconv.ParseFloat(ev.Transaction.Fee.String(), 64)

	obs := make([]NetworkObservationFeature, len(ev.Observations))
	for i, o := range ev.Observations {
		obs[i] = NetworkObservationFeature{
			ObservationID: o.ObservationID,
			ObservedAt:    o.ObservedAt.UTC().Format(time.RFC3339),
			SrcIP:         o.SrcIP.String(),
			GeoCountry:    o.GeoCountry,
			ASN:           o.ASN,
		}
	}

	return TransactionFeatures{
		TXID:                ev.Transaction.TXID,
		AmountBTC:           amountBTC,
		FeeBTC:              feeBTC,
		InputCount:          len(ev.Transaction.InputAddresses),
		OutputCount:         len(ev.Transaction.OutputAddresses),
		ScriptType:          string(ev.Transaction.ScriptType),
		InputAddresses:      append([]string(nil), ev.Transaction.InputAddresses...),
		OutputAddresses:     append([]string(nil), ev.Transaction.OutputAddresses...),
		InputAmounts:        satoshisToBTC(ev.Transaction.InputAmounts),
		OutputAmounts:       satoshisToBTC(ev.Transaction.OutputAmounts),
		NetworkObservations: obs,
	}
}

func satoshisToBTC(amounts []domain.Satoshis) []float64 {
	out := make([]float64, len(amounts))
	for i, amount := range amounts {
		out[i], _ = strconv.ParseFloat(amount.String(), 64)
	}
	return out
}
