// internal/ingestion/network_test.go
package ingestion

import (
	"strings"
	"testing"

	"github.com/dipak0000812/TraceLayer/internal/domain"
)

const netHeader = "observation_id,observed_txid,timestamp,src_ip,dst_ip,src_port,dst_port,provenance,dataset_id,generator_version\n"

func TestParseNetworkObservationsCSV_ValidRow(t *testing.T) {
	row := `OBS-aaaaaaaaaaaaaaaaaaaaaaaa,b85038db8d34756615a8c82752b931ded57f99c1911097b5d0e21f0c648e638b,2025-03-22T20:27:05Z,203.0.113.10,198.51.100.20,18333,8333,SYNTHETIC,6bc084b677a63411,1.0.0` + "\n"

	valid, rejected, err := ParseNetworkObservationsCSV(strings.NewReader(netHeader + row))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(rejected) != 0 {
		t.Fatalf("expected 0 rejections, got %d: %+v", len(rejected), rejected)
	}
	if len(valid) != 1 {
		t.Fatalf("expected 1 valid observation, got %d", len(valid))
	}
	if valid[0].SrcPort != 18333 {
		t.Errorf("SrcPort = %d, want 18333", valid[0].SrcPort)
	}
	if valid[0].CorrelationStatus != domain.CorrelationPending {
		t.Errorf("CorrelationStatus = %s, want PENDING", valid[0].CorrelationStatus)
	}
	if err := valid[0].Validate(); err != nil {
		t.Errorf("parsed observation failed Validate(): %v", err)
	}
}

func TestParseNetworkObservationsCSV_BadRowRejectedNotFatal(t *testing.T) {
	badRow := `OBS-bbbbbbbbbbbbbbbbbbbbbbbb,b85038db8d34756615a8c82752b931ded57f99c1911097b5d0e21f0c648e638b,NOT-A-TIME,203.0.113.10,198.51.100.20,18333,8333,SYNTHETIC,ds,1.0.0` + "\n"
	goodRow := `OBS-cccccccccccccccccccccccc,9e752d5ae640e33b1ea420bb782e7b59617465516801b8d0599378e4694917d2,2025-01-07T23:29:00Z,203.0.113.11,198.51.100.21,18333,8333,SYNTHETIC,6bc084b677a63411,1.0.0` + "\n"

	valid, rejected, err := ParseNetworkObservationsCSV(strings.NewReader(netHeader + badRow + goodRow))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(valid) != 1 {
		t.Fatalf("expected the good row to still parse: got %d valid", len(valid))
	}
	if len(rejected) != 1 || rejected[0].FieldName != "timestamp" || rejected[0].LineNumber != 2 {
		t.Fatalf("unexpected rejection: %+v", rejected)
	}
}

func TestParseNetworkObservationsCSV_DuplicateObservationIDsNotDeduped(t *testing.T) {
	row := `OBS-dddddddddddddddddddddddd,b85038db8d34756615a8c82752b931ded57f99c1911097b5d0e21f0c648e638b,2025-03-22T20:27:05Z,203.0.113.10,198.51.100.20,18333,8333,SYNTHETIC,ds,1.0.0` + "\n"

	valid, _, err := ParseNetworkObservationsCSV(strings.NewReader(netHeader + row + row))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(valid) != 2 {
		t.Fatalf("parser must not dedupe — expected 2 valid rows (dedup is storage's job), got %d", len(valid))
	}
}