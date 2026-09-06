// internal/domain/transaction_test.go
package domain

import (
	"errors"
	"testing"
	"time"
)

func validSeed42Tx() Transaction {
	// Exact values from the seed-42 sample row in transactions.csv.
	return Transaction{
		TXID:             "b85038db8d34756615a8c82752b931ded57f99c1911097b5d0e21f0c648e638b",
		BlockTime:        time.Now(),
		InputAddresses:   []string{"sbc18d20c1d613b34c0e6946f41fc34692fc9daf10"},
		OutputAddresses:  []string{"sbc1231363128bc87877088348b38c25cc78ac7f16"},
		InputAmounts:     []Satoshis{74038447},
		OutputAmounts:    []Satoshis{73932792},
		Fee:              105655,
		ScriptType:       ScriptTypeP2PKH,
		Provenance:       ProvenanceSynthetic,
		DatasetID:        "6bc084b677a63411",
		GeneratorVersion: "1.0.0",
	}
}

func TestTransaction_Validate_Valid(t *testing.T) {
	if err := validSeed42Tx().Validate(); err != nil {
		t.Fatalf("expected valid seed-42 transaction, got err: %v", err)
	}
}

func TestTransaction_Validate_FeeMismatch(t *testing.T) {
	tx := validSeed42Tx()
	tx.Fee = 999999 // wrong on purpose
	err := tx.Validate()
	if !errors.Is(err, ErrFeeMismatch) {
		t.Fatalf("expected ErrFeeMismatch, got: %v", err)
	}
}

func TestTransaction_Validate_BadTXID(t *testing.T) {
	tx := validSeed42Tx()
	tx.TXID = "not-a-txid"
	if err := tx.Validate(); !errors.Is(err, ErrInvalidTXID) {
		t.Fatalf("expected ErrInvalidTXID, got: %v", err)
	}
}

func TestTransaction_Validate_AmountCountMismatch(t *testing.T) {
	tx := validSeed42Tx()
	tx.InputAmounts = append(tx.InputAmounts, 1)
	if err := tx.Validate(); !errors.Is(err, ErrAmountCountMismatch) {
		t.Fatalf("expected ErrAmountCountMismatch, got: %v", err)
	}
}

func TestTransaction_Validate_UnknownScriptType(t *testing.T) {
	tx := validSeed42Tx()
	tx.ScriptType = "NOT_A_TYPE"
	if err := tx.Validate(); !errors.Is(err, ErrInvalidScriptType) {
		t.Fatalf("expected ErrInvalidScriptType, got: %v", err)
	}
}