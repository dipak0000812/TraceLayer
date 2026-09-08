// internal/intelligence/client_test.go
package intelligence

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// This fixture is the concrete example payload for Sayali — the exact
// shape this client sends and expects back, lifted straight from
// docs/API_CONTRACT.md §3.
const stubResponseBody = `{
  "scores": [
    {
      "txid": "b85038db8d34756615a8c82752b931ded57f99c1911097b5d0e21f0c648e638b",
      "chain_score": 0.42,
      "network_score": 0.88,
      "mixing_penalty": 0.0,
      "network_quality_q": 0.94,
      "fused_score": 0.79,
      "heuristic_association_strength": 0.81,
      "flags": ["RAPID_DISPERSION"],
      "explanation": "High peer dispersion with rapid broadcast across Latin America."
    }
  ]
}`

func testRequest() ScoreRequest {
	return ScoreRequest{Transactions: []TransactionFeatures{{
		TXID:        "b85038db8d34756615a8c82752b931ded57f99c1911097b5d0e21f0c648e638b",
		AmountBTC:   0.74038447,
		FeeBTC:      0.00105655,
		InputCount:  1,
		OutputCount: 1,
		ScriptType:  "P2PKH",
	}}}
}

func TestClient_Score_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/intelligence/score" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(stubResponseBody))
	}))
	defer server.Close()

	client := NewClient(server.URL, time.Second)
	resp, err := client.Score(context.Background(), testRequest())
	if err != nil {
		t.Fatalf("Score: %v", err)
	}
	if len(resp.Scores) != 1 {
		t.Fatalf("expected 1 score, got %d", len(resp.Scores))
	}
	if resp.Scores[0].FusedScore != 0.79 {
		t.Errorf("FusedScore = %v, want 0.79", resp.Scores[0].FusedScore)
	}
}

func TestClient_Score_WorkerReturns500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"model failed"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, time.Second)
	_, err := client.Score(context.Background(), testRequest())
	if err == nil || !strings.Contains(err.Error(), "worker unavailable") {
		t.Fatalf("expected ErrWorkerUnavailable, got: %v", err)
	}
	var workerErr *WorkerError
	if !errors.As(err, &workerErr) || workerErr.Kind != WorkerHTTPError {
		t.Fatalf("expected HTTP worker error, got: %v", err)
	}
}

func TestClient_Score_MalformedResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	client := NewClient(server.URL, time.Second)
	_, err := client.Score(context.Background(), testRequest())
	if err == nil || !strings.Contains(err.Error(), "worker unavailable") {
		t.Fatalf("expected ErrWorkerUnavailable, got: %v", err)
	}
	var workerErr *WorkerError
	if !errors.As(err, &workerErr) || workerErr.Kind != WorkerMalformed {
		t.Fatalf("expected malformed worker error, got: %v", err)
	}
}

func TestClient_Score_Unreachable_FailsFastNotHang(t *testing.T) {
	client := NewClient("http://127.0.0.1:1", 2*time.Second) // port 1: nothing ever listens here
	start := time.Now()
	_, err := client.Score(context.Background(), testRequest())
	if err == nil {
		t.Fatal("expected error against unreachable worker")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("took %s — client timeout isn't bounding the call", elapsed)
	}
	var workerErr *WorkerError
	if !errors.As(err, &workerErr) || workerErr.Kind != WorkerUnavailable {
		t.Fatalf("expected unavailable worker error, got: %v", err)
	}
}

func TestClient_Score_RejectsMismatchedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"scores":[]}`))
	}))
	defer server.Close()

	_, err := NewClient(server.URL, time.Second).Score(context.Background(), testRequest())
	var workerErr *WorkerError
	if !errors.As(err, &workerErr) || workerErr.Kind != WorkerMalformed {
		t.Fatalf("expected malformed worker error, got: %v", err)
	}
}

func TestClient_Score_RejectsEmptyRequest(t *testing.T) {
	_, err := NewClient("http://127.0.0.1:1", time.Second).Score(context.Background(), ScoreRequest{})
	if err == nil || !strings.Contains(err.Error(), "at least one transaction") {
		t.Fatalf("expected empty request validation error, got: %v", err)
	}
}

func TestClient_Score_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()

	_, err := NewClient(server.URL, 10*time.Millisecond).Score(context.Background(), testRequest())
	var workerErr *WorkerError
	if !errors.As(err, &workerErr) || workerErr.Kind != WorkerTimeout {
		t.Fatalf("expected timeout worker error, got: %v", err)
	}
}

func TestRealWorkerScore(t *testing.T) {
	workerURL := os.Getenv("INTELLIGENCE_URL")
	if workerURL == "" {
		t.Skip("INTELLIGENCE_URL is not set; set it to run against the real FastAPI worker")
	}

	request := testRequest()
	request.Transactions[0].InputAddresses = []string{"sbc1input"}
	request.Transactions[0].OutputAddresses = []string{"sbc1output"}
	request.Transactions[0].InputAmounts = []float64{0.74038447}
	request.Transactions[0].OutputAmounts = []float64{0.73932792}
	request.Transactions[0].NetworkObservations = []NetworkObservationFeature{{
		ObservationID: "OBS-real-worker",
		ObservedAt:    "2025-03-22T16:34:25Z",
		SrcIP:         "198.51.100.98",
	}}

	resp, err := NewClient(workerURL, 10*time.Second).Score(context.Background(), request)
	if err != nil {
		t.Fatalf("real worker Score: %v", err)
	}
	if len(resp.Scores) != 1 || resp.Scores[0].TXID != request.Transactions[0].TXID {
		t.Fatalf("unexpected real worker response: %+v", resp)
	}
	if len(resp.Scores[0].TopFeatures) == 0 || resp.Scores[0].Explanation == "" {
		t.Fatalf("real worker did not return explanations: %+v", resp.Scores[0])
	}
}
