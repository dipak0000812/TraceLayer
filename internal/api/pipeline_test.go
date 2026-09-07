package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// truncateAllTables wipes every table this test touches. Integration tests
// must not depend on the database happening to be clean.
func truncateAllTables(t *testing.T, deps Deps) {
	t.Helper()
	tables := []string{"forensic_leads", "entities", "network_observations", "transactions"}
	for _, tbl := range tables {
		if _, err := deps.Pool.Exec(context.Background(), "TRUNCATE TABLE "+tbl+" CASCADE"); err != nil {
			t.Fatalf("truncating %s: %v", tbl, err)
		}
	}
}

const testTxCSV = "txid,timestamp,input_addresses,output_addresses,input_amounts,output_amounts,fee,script_type,provenance,dataset_id,generator_version\n" +
	`b85038db8d34756615a8c82752b931ded57f99c1911097b5d0e21f0c648e638b,2025-03-22T20:27:00Z,"[""sbc18d20c1d613b34c0e6946f41fc34692fc9daf10""]","[""sbc1out1""]",[0.74038447],[0.73932792],0.00105655,P2PKH,SYNTHETIC,6bc084b677a63411,1.0.0` + "\n"

const testNetCSV = "observation_id,txid,timestamp,src_ip,dst_ip,src_port,dst_port,provenance,dataset_id,generator_version\n" +
	`OBS-aaaaaaaaaaaaaaaaaaaaaaaa,b85038db8d34756615a8c82752b931ded57f99c1911097b5d0e21f0c648e638b,2025-03-22T20:27:05Z,203.0.113.10,198.51.100.20,18333,8333,SYNTHETIC,6bc084b677a63411,1.0.0` + "\n"

func doHandler(t *testing.T, handler http.HandlerFunc, method, path, body string) (*httptest.ResponseRecorder, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "text/csv")
	rec := httptest.NewRecorder()
	handler(rec, req)

	var parsed map[string]interface{}
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
			t.Fatalf("response body isn't valid JSON: %v; body=%s", err, rec.Body.String())
		}
	}
	return rec, parsed
}

// TestPipeline_IngestCorrelateLeads_EndToEnd exercises the HTTP handler chain
// with real CSV bodies and verifies the response shapes and persisted quality.
func TestPipeline_IngestCorrelateLeads_EndToEnd(t *testing.T) {
	deps := testDeps(t)
	truncateAllTables(t, deps)
	t.Cleanup(func() { truncateAllTables(t, deps) })

	txRec, txBody := doHandler(t, ingestBlockchainHandler(deps), http.MethodPost, "/api/v1/ingest/blockchain", testTxCSV)
	if txRec.Code != http.StatusOK {
		t.Fatalf("ingest blockchain status = %d, want 200; body=%v", txRec.Code, txBody)
	}
	if got, _ := txBody["ingested_count"].(float64); got != 1 {
		t.Fatalf("ingested_count = %v, want 1; body=%v", txBody["ingested_count"], txBody)
	}
	if got, _ := txBody["rejected_count"].(float64); got != 0 {
		t.Fatalf("rejected_count = %v, want 0; body=%v", txBody["rejected_count"], txBody)
	}

	netRec, netBody := doHandler(t, ingestNetworkHandler(deps), http.MethodPost, "/api/v1/ingest/network", testNetCSV)
	if netRec.Code != http.StatusOK {
		t.Fatalf("ingest network status = %d, want 200; body=%v", netRec.Code, netBody)
	}
	if got, _ := netBody["ingested_count"].(float64); got != 1 {
		t.Fatalf("ingested_count = %v, want 1; body=%v", netBody["ingested_count"], netBody)
	}

	corrRec, corrBody := doHandler(t, correlateHandler(deps), http.MethodPost, "/api/v1/correlate", "")
	if corrRec.Code != http.StatusOK {
		t.Fatalf("correlate status = %d, want 200; body=%v", corrRec.Code, corrBody)
	}
	if got, _ := corrBody["observations_correlated"].(float64); got != 1 {
		t.Fatalf("observations_correlated = %v, want 1; body=%v", corrBody["observations_correlated"], corrBody)
	}
	if got, _ := corrBody["entities_clustered"].(float64); got != 1 {
		t.Fatalf("entities_clustered = %v, want 1; body=%v", corrBody["entities_clustered"], corrBody)
	}
	if got, _ := corrBody["leads_generated"].(float64); got != 1 {
		t.Fatalf("leads_generated = %v, want 1; body=%v", corrBody["leads_generated"], corrBody)
	}

	leadsRec, leadsBody := doHandler(t, leadsListHandler(deps), http.MethodGet, "/api/v1/leads?limit=5", "")
	if leadsRec.Code != http.StatusOK {
		t.Fatalf("leads list status = %d, want 200; body=%v", leadsRec.Code, leadsBody)
	}
	leads, ok := leadsBody["leads"].([]interface{})
	if !ok || len(leads) != 1 {
		t.Fatalf("expected 1 lead in response, got %v; body=%v", leadsBody["leads"], leadsBody)
	}
	lead := leads[0].(map[string]interface{})
	if lead["primary_txid"] != "b85038db8d34756615a8c82752b931ded57f99c1911097b5d0e21f0c648e638b" {
		t.Fatalf("primary_txid = %v, want the test fixture's txid", lead["primary_txid"])
	}
	quality, _ := lead["network_evidence_quality"].(float64)
	if quality <= 0 {
		t.Fatalf("network_evidence_quality = %v, want > 0 given 1 correlated observation", quality)
	}
}
