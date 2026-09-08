// internal/ingestion/network.go
package ingestion

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/netip"
	"strconv"
	"time"

	"github.com/dipak0000812/TraceLayer/internal/domain"
)

var requiredNetworkColumns = []string{
	"observation_id", "txid", "timestamp", "src_ip", "dst_ip",
	"src_port", "dst_port", "provenance", "dataset_id", "generator_version",
}

// ParseNetworkObservationsCSV streams network_observations.csv per
// DATA_CONTRACT.md §2.2. txid is NOT checked against the
// transactions table here — network ingestion is decoupled from blockchain
// ingestion by design; that join is the correlation engine's job (Phase 5).
// Malformed/invalid rows are collected as rejections, matching the
// transaction parser's partial-batch tolerance. Duplicate observation_ids
// are NOT deduplicated here — that's the storage layer's job.
func ParseNetworkObservationsCSV(r io.Reader) (valid []domain.NetworkObservation, rejections []RowRejection, err error) {
	r, err = normalizeInput(r, requiredNetworkColumns)
	if err != nil {
		return nil, nil, err
	}
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1

	header, err := cr.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("ingestion: reading header row: %w", err)
	}
	col, err := columnIndex(header, requiredNetworkColumns)
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

		obs, field, parseErr := parseNetworkObservationRow(record, col)
		if parseErr != nil {
			rejections = append(rejections, RowRejection{
				LineNumber: lineNumber, Reason: parseErr.Error(), FieldName: field, RawRecord: joinRecord(record),
			})
			continue
		}
		if valErr := obs.Validate(); valErr != nil {
			rejections = append(rejections, RowRejection{
				LineNumber: lineNumber, Reason: valErr.Error(), RawRecord: joinRecord(record),
			})
			continue
		}
		valid = append(valid, obs)
	}
	return valid, rejections, nil
}

func parseNetworkObservationRow(record []string, col map[string]int) (domain.NetworkObservation, string, error) {
	get := func(name string) string {
		if i, ok := col[name]; ok && i < len(record) {
			return record[i]
		}
		return ""
	}

	tsStr := get("timestamp")
	observedAt, err := time.Parse(time.RFC3339, tsStr)
	if err != nil {
		return domain.NetworkObservation{}, "timestamp", fmt.Errorf("VALIDATION_FAILED: invalid timestamp %q", tsStr)
	}

	srcIP, err := netip.ParseAddr(get("src_ip"))
	if err != nil {
		return domain.NetworkObservation{}, "src_ip", fmt.Errorf("VALIDATION_FAILED: invalid src_ip %q", get("src_ip"))
	}
	dstIP, err := netip.ParseAddr(get("dst_ip"))
	if err != nil {
		return domain.NetworkObservation{}, "dst_ip", fmt.Errorf("VALIDATION_FAILED: invalid dst_ip %q", get("dst_ip"))
	}

	srcPort, field, err := parsePort(get("src_port"), "src_port")
	if err != nil {
		return domain.NetworkObservation{}, field, err
	}
	dstPort, field, err := parsePort(get("dst_port"), "dst_port")
	if err != nil {
		return domain.NetworkObservation{}, field, err
	}

	return domain.NetworkObservation{
		ObservationID:     get("observation_id"),
		ObservedTXID:      get("txid"),
		ObservedAt:        observedAt,
		SrcIP:             srcIP,
		DstIP:             dstIP,
		SrcPort:           srcPort,
		DstPort:           dstPort,
		CorrelationStatus: domain.CorrelationPending,
		Provenance:        domain.Provenance(get("provenance")),
		DatasetID:         get("dataset_id"),
		GeneratorVersion:  get("generator_version"),
	}, "", nil
}

func parsePort(raw, field string) (uint16, string, error) {
	p, err := strconv.ParseUint(raw, 10, 16)
	if err != nil {
		return 0, field, fmt.Errorf("VALIDATION_FAILED: invalid %s %q", field, raw)
	}
	return uint16(p), "", nil
}
