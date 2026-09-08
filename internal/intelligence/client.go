// internal/intelligence/client.go
package intelligence

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"time"
)

// ErrWorkerUnavailable covers every failure mode talking to the Python
// intelligence worker — network error, non-2xx, or an unparsable body. Per
// the project brief: "If Python is unavailable, the system must explicitly
// report FAILED/INCOMPLETE rather than pretending detection succeeded."
// Callers must propagate this, never substitute a default/zero score.
var ErrWorkerUnavailable = errors.New("intelligence: worker unavailable")

type WorkerFailureKind string

const (
	WorkerUnavailable WorkerFailureKind = "unavailable"
	WorkerTimeout     WorkerFailureKind = "timeout"
	WorkerHTTPError   WorkerFailureKind = "http_error"
	WorkerMalformed   WorkerFailureKind = "malformed_response"
)

type WorkerError struct {
	Kind       WorkerFailureKind
	StatusCode int
	Cause      error
}

func (e *WorkerError) Error() string {
	if e.StatusCode != 0 {
		return fmt.Sprintf("intelligence worker %s (status %d): %v", e.Kind, e.StatusCode, e.Cause)
	}
	return fmt.Sprintf("intelligence worker %s: %v", e.Kind, e.Cause)
}

func (e *WorkerError) Unwrap() error { return e.Cause }

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{baseURL: baseURL, http: &http.Client{Timeout: timeout}}
}

// ScoreRequest/Response mirror docs/API_CONTRACT.md §3 exactly.
type ScoreRequest struct {
	Transactions []TransactionFeatures `json:"transactions"`
	Explainer    string                `json:"explainer,omitempty"`
}

type TransactionFeatures struct {
	TXID                string                      `json:"txid"`
	AmountBTC           float64                     `json:"amount_btc"`
	FeeBTC              float64                     `json:"fee_btc"`
	InputCount          int                         `json:"input_count"`
	OutputCount         int                         `json:"output_count"`
	ScriptType          string                      `json:"script_type"`
	InputAddresses      []string                    `json:"input_addresses,omitempty"`
	OutputAddresses     []string                    `json:"output_addresses,omitempty"`
	InputAmounts        []float64                   `json:"input_amounts,omitempty"`
	OutputAmounts       []float64                   `json:"output_amounts,omitempty"`
	NetworkObservations []NetworkObservationFeature `json:"network_observations"`
}

type NetworkObservationFeature struct {
	ObservationID string  `json:"observation_id"`
	ObservedAt    string  `json:"observed_at"`
	SrcIP         string  `json:"src_ip"`
	GeoCountry    *string `json:"geo_country"`
	ASN           *string `json:"asn"`
}

type ScoreResponse struct {
	Scores []TransactionScore `json:"scores"`
}

type TransactionScore struct {
	TXID                         string               `json:"txid"`
	ChainScore                   float64              `json:"chain_score"`
	NetworkScore                 float64              `json:"network_score"`
	MixingPenalty                float64              `json:"mixing_penalty"`
	NetworkQualityQ              float64              `json:"network_quality_q"`
	FusedScore                   float64              `json:"fused_score"`
	HeuristicAssociationStrength float64              `json:"heuristic_association_strength"`
	Flags                        []string             `json:"flags"`
	Explanation                  string               `json:"explanation"`
	TopFeatures                  []FeatureAttribution `json:"top_features"`
	ExplainabilityMethod         string               `json:"explainability_method"`
}

type FeatureAttribution struct {
	Feature   string  `json:"feature"`
	SHAPValue float64 `json:"shap_value"`
	Direction string  `json:"direction"`
}

// Score calls POST {baseURL}/intelligence/score. Any failure returns
// ErrWorkerUnavailable wrapped with detail — no partial/default scores are
// ever fabricated here.
func (c *Client) Score(ctx context.Context, req ScoreRequest) (ScoreResponse, error) {
	if len(req.Transactions) == 0 {
		return ScoreResponse{}, errors.New("intelligence: score request must contain at least one transaction")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return ScoreResponse{}, fmt.Errorf("intelligence: encoding request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/intelligence/score", bytes.NewReader(body))
	if err != nil {
		return ScoreResponse{}, fmt.Errorf("intelligence: building request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		kind := WorkerUnavailable
		var netErr net.Error
		if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
			kind = WorkerTimeout
		}
		return ScoreResponse{}, &WorkerError{Kind: kind, Cause: fmt.Errorf("%w: %v", ErrWorkerUnavailable, err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return ScoreResponse{}, &WorkerError{
			Kind:       WorkerHTTPError,
			StatusCode: resp.StatusCode,
			Cause:      fmt.Errorf("%w: status %d: %s", ErrWorkerUnavailable, resp.StatusCode, respBody),
		}
	}

	var out ScoreResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return ScoreResponse{}, &WorkerError{Kind: WorkerMalformed, Cause: fmt.Errorf("%w: decoding response: %v", ErrWorkerUnavailable, err)}
	}
	if err := validateScoreResponse(req, out); err != nil {
		return ScoreResponse{}, &WorkerError{Kind: WorkerMalformed, Cause: fmt.Errorf("%w: invalid response: %v", ErrWorkerUnavailable, err)}
	}
	return out, nil
}

func validateScoreResponse(req ScoreRequest, resp ScoreResponse) error {
	if len(resp.Scores) != len(req.Transactions) {
		return fmt.Errorf("got %d scores for %d transactions", len(resp.Scores), len(req.Transactions))
	}
	for i, score := range resp.Scores {
		if score.TXID != req.Transactions[i].TXID {
			return fmt.Errorf("score %d has txid %q, want %q", i, score.TXID, req.Transactions[i].TXID)
		}
		for name, value := range map[string]float64{
			"chain_score":                    score.ChainScore,
			"network_score":                  score.NetworkScore,
			"mixing_penalty":                 score.MixingPenalty,
			"network_quality_q":              score.NetworkQualityQ,
			"fused_score":                    score.FusedScore,
			"heuristic_association_strength": score.HeuristicAssociationStrength,
		} {
			if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 1 {
				return fmt.Errorf("score %d field %s is outside [0,1]", i, name)
			}
		}
	}
	return nil
}
