// internal/ingestion/transactions.go
package ingestion

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/dipak0000812/TraceLayer/internal/domain"
)

// RowRejection is one failed CSV row, shaped to become a rejection_logs row
// once Phase 7 wires this into a batch-tracked HTTP endpoint. This package
// only returns rejections in memory — it never writes to Postgres.
type RowRejection struct {
	LineNumber int
	Reason     string
	FieldName  string
	RawRecord  string
}

var requiredColumns = []string{
	"txid", "timestamp", "input_addresses", "output_addresses",
	"input_amounts", "output_amounts", "fee", "script_type",
	"provenance", "dataset_id", "generator_version",
}

// ParseTransactionsCSV streams transactions.csv per DATA_CONTRACT.md section
// 2.1. Malformed rows are collected as rejections and parsing continues —
// per API_CONTRACT.md's partial batch tolerance, one bad row never aborts
// the batch. Duplicate txids across rows are NOT deduplicated here; that
// decision needs the database and belongs to Phase 3b.
func ParseTransactionsCSV(r io.Reader) (valid []domain.Transaction, rejections []RowRejection, err error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1

	header, err := cr.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("ingestion: reading header row: %w", err)
	}
	col, err := columnIndex(header, requiredColumns)
	if err != nil {
		return nil, nil, fmt.Errorf("ingestion: %w", err)
	}

	lineNumber := 1
	for {
		record, readErr := cr.Read()
		if readErr == io.EOF {
			break
		}
		lineNumber++
		if readErr != nil {
			rejections = append(rejections, RowRejection{
				LineNumber: lineNumber, Reason: "MALFORMED_CSV_ROW", RawRecord: readErr.Error(),
			})
			continue
		}

		tx, field, parseErr := parseTransactionRow(record, col)
		if parseErr != nil {
			rejections = append(rejections, RowRejection{
				LineNumber: lineNumber, Reason: parseErr.Error(), FieldName: field, RawRecord: joinRecord(record),
			})
			continue
		}
		if valErr := tx.Validate(); valErr != nil {
			rejections = append(rejections, RowRejection{
				LineNumber: lineNumber, Reason: valErr.Error(), RawRecord: joinRecord(record),
			})
			continue
		}
		valid = append(valid, tx)
	}
	return valid, rejections, nil
}

func parseTransactionRow(record []string, col map[string]int) (domain.Transaction, string, error) {
	get := func(name string) string {
		if i, ok := col[name]; ok && i < len(record) {
			return record[i]
		}
		return ""
	}

	tsStr := get("timestamp")
	blockTime, err := time.Parse(time.RFC3339, tsStr)
	if err != nil {
		return domain.Transaction{}, "timestamp", fmt.Errorf("VALIDATION_FAILED: invalid timestamp %q", tsStr)
	}

	var inputAddrs, outputAddrs []string
	if err := json.Unmarshal([]byte(get("input_addresses")), &inputAddrs); err != nil {
		return domain.Transaction{}, "input_addresses", fmt.Errorf("VALIDATION_FAILED: invalid input_addresses JSON")
	}
	if err := json.Unmarshal([]byte(get("output_addresses")), &outputAddrs); err != nil {
		return domain.Transaction{}, "output_addresses", fmt.Errorf("VALIDATION_FAILED: invalid output_addresses JSON")
	}

	inputAmounts, field, err := parseAmountArray(get("input_amounts"), "input_amounts")
	if err != nil {
		return domain.Transaction{}, field, err
	}
	outputAmounts, field, err := parseAmountArray(get("output_amounts"), "output_amounts")
	if err != nil {
		return domain.Transaction{}, field, err
	}

	fee, err := domain.ParseBTCString(get("fee"))
	if err != nil {
		return domain.Transaction{}, "fee", fmt.Errorf("VALIDATION_FAILED: invalid fee: %v", err)
	}

	return domain.Transaction{
		TXID:             get("txid"),
		BlockTime:        blockTime,
		InputAddresses:   inputAddrs,
		OutputAddresses:  outputAddrs,
		InputAmounts:     inputAmounts,
		OutputAmounts:    outputAmounts,
		Fee:              fee,
		ScriptType:       domain.ScriptType(get("script_type")),
		Provenance:       domain.Provenance(get("provenance")),
		DatasetID:        get("dataset_id"),
		GeneratorVersion: get("generator_version"),
	}, "", nil
}

func parseAmountArray(raw, field string) ([]domain.Satoshis, string, error) {
	var nums []json.Number
	if err := json.Unmarshal([]byte(raw), &nums); err != nil {
		return nil, field, fmt.Errorf("VALIDATION_FAILED: invalid %s JSON", field)
	}
	out := make([]domain.Satoshis, len(nums))
	for i, n := range nums {
		s, err := domain.ParseBTCString(n.String())
		if err != nil {
			return nil, field, fmt.Errorf("VALIDATION_FAILED: invalid amount in %s: %v", field, err)
		}
		out[i] = s
	}
	return out, "", nil
}

func columnIndex(header, required []string) (map[string]int, error) {
	idx := make(map[string]int, len(header))
	for i, h := range header {
		idx[h] = i
	}
	for _, name := range required {
		if _, ok := idx[name]; !ok {
			return nil, fmt.Errorf("missing required column %q in header", name)
		}
	}
	return idx, nil
}

func joinRecord(record []string) string {
	b, _ := json.Marshal(record)
	return string(b)
}