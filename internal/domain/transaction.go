// internal/domain/transaction.go
package domain

import (
	"errors"
	"fmt"
	"time"
)

// Transaction is DATA_CONTRACT.md section 2.1. All amounts are exact
// Satoshis.
type Transaction struct {
	TXID             string
	BlockTime        time.Time
	InputAddresses   []string
	OutputAddresses  []string
	InputAmounts     []Satoshis
	OutputAmounts    []Satoshis
	Fee              Satoshis
	ScriptType       ScriptType
	Provenance       Provenance
	DatasetID        string
	GeneratorVersion string
}

// TransactionInput/TransactionOutput are read-side view types reconstructed
// from a Transaction's address/amount arrays (which persist as JSONB per
// DATA_CONTRACT.md's DDL, not as separate tables — see the note above this
// phase). They exist for API response construction, not for direct
// persistence.
type TransactionInput struct {
	Index   int
	Address string
	Value   Satoshis
}

type TransactionOutput struct {
	Index      int
	Address    string
	Value      Satoshis
	ScriptType ScriptType
}

var (
	ErrInvalidTXID          = errors.New("domain: txid must be 64 lowercase hex characters")
	ErrEmptyInputAddresses  = errors.New("domain: transaction must have at least one input address")
	ErrEmptyOutputAddresses = errors.New("domain: transaction must have at least one output address")
	ErrAmountCountMismatch  = errors.New("domain: amount array length does not match address array length")
	ErrInvalidScriptType    = errors.New("domain: unrecognized script_type")
	ErrInvalidProvenance    = errors.New("domain: unrecognized provenance")
	ErrFeeMismatch          = errors.New("domain: fee does not equal sum(inputs) - sum(outputs)")
)

// Validate enforces DATA_CONTRACT.md section 2.1's structural and
// cross-field invariants, including the fee-balance rule:
// fee == sum(input_amounts) - sum(output_amounts) >= 0.
// It does not touch a database or any other transaction — duplicate/conflict
// handling belongs to Phase 3's ingestion layer, not here.
func (t Transaction) Validate() error {
	if !ValidTXID(t.TXID) {
		return fmt.Errorf("%w: %q", ErrInvalidTXID, t.TXID)
	}
	if len(t.InputAddresses) == 0 {
		return ErrEmptyInputAddresses
	}
	if len(t.OutputAddresses) == 0 {
		return ErrEmptyOutputAddresses
	}
	if len(t.InputAddresses) != len(t.InputAmounts) {
		return fmt.Errorf("%w: %d input addresses vs %d input amounts",
			ErrAmountCountMismatch, len(t.InputAddresses), len(t.InputAmounts))
	}
	if len(t.OutputAddresses) != len(t.OutputAmounts) {
		return fmt.Errorf("%w: %d output addresses vs %d output amounts",
			ErrAmountCountMismatch, len(t.OutputAddresses), len(t.OutputAmounts))
	}
	if !t.ScriptType.Valid() {
		return fmt.Errorf("%w: %q", ErrInvalidScriptType, t.ScriptType)
	}
	if !t.Provenance.Valid() {
		return fmt.Errorf("%w: %q", ErrInvalidProvenance, t.Provenance)
	}

	var inputSum, outputSum Satoshis
	for _, a := range t.InputAmounts {
		if a < 0 {
			return ErrNegativeAmount
		}
		inputSum += a
	}
	for _, a := range t.OutputAmounts {
		if a < 0 {
			return ErrNegativeAmount
		}
		outputSum += a
	}
	if t.Fee < 0 {
		return ErrNegativeAmount
	}
	if inputSum-outputSum != t.Fee {
		return fmt.Errorf("%w: inputs=%s outputs=%s fee=%s",
			ErrFeeMismatch, inputSum, outputSum, t.Fee)
	}
	return nil
}