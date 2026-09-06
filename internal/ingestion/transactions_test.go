// internal/ingestion/transactions_test.go
package ingestion

import (
	"strings"
	"testing"
)

const txHeader = "txid,timestamp,input_addresses,output_addresses,input_amounts,output_amounts,fee,script_type,provenance,dataset_id,generator_version\n"

func TestParseTransactionsCSV_ValidRow(t *testing.T) {
	row := `b85038db8d34756615a8c82752b931ded57f99c1911097b5d0e21f0c648e638b,2025-03-22T20:27:00Z,"[""sbc18d20c1d613b34c0e6946f41fc34692fc9daf10""]","[""sbc1231363128bc87877088348b38c25cc78ac7f16""]",[0.74038447],[0.73932792],0.00105655,P2PKH,SYNTHETIC,6bc084b677a63411,1.0.0` + "\n"

	valid, rejected, err := ParseTransactionsCSV(strings.NewReader(txHeader + row))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(rejected) != 0 {
		t.Fatalf("expected 0 rejections, got %d: %+v", len(rejected), rejected)
	}
	if len(valid) != 1 {
		t.Fatalf("expected 1 valid transaction, got %d", len(valid))
	}
	if valid[0].Fee != 105655 {
		t.Errorf("Fee = %d, want 105655", valid[0].Fee)
	}
	if err := valid[0].Validate(); err != nil {
		t.Errorf("parsed transaction failed Validate(): %v", err)
	}
}

func TestParseTransactionsCSV_BadRowRejectedNotFatal(t *testing.T) {
	badRow := `b85038db8d34756615a8c82752b931ded57f99c1911097b5d0e21f0c648e638b,NOT-A-TIME,"[""a""]","[""b""]",[1],[1],0,P2PKH,SYNTHETIC,ds,1.0.0` + "\n"
	goodRow := `9e752d5ae640e33b1ea420bb782e7b59617465516801b8d0599378e4694917d2,2025-01-07T23:29:00Z,"[""sbc1f8c73121f5076119964ac63d2799ea8f09896b""]","[""sbc1816c171b1d2473905a460ce504835f6e8794ce""]",[1.028643],[1.02741318],0.00122982,P2TR,SYNTHETIC,6bc084b677a63411,1.0.0` + "\n"

	valid, rejected, err := ParseTransactionsCSV(strings.NewReader(txHeader + badRow + goodRow))
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

func TestParseTransactionsCSV_DuplicateTXIDsNotDeduped(t *testing.T) {
	row := `b85038db8d34756615a8c82752b931ded57f99c1911097b5d0e21f0c648e638b,2025-03-22T20:27:00Z,"[""a""]","[""b""]",[1],[1],0,P2PKH,SYNTHETIC,ds,1.0.0` + "\n"

	valid, _, err := ParseTransactionsCSV(strings.NewReader(txHeader + row + row))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(valid) != 2 {
		t.Fatalf("parser must not dedupe — expected 2 valid rows (dedup is Phase 3b's job), got %d", len(valid))
	}
}