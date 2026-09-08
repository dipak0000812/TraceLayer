package ingestion

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
)

func normalizeInput(r io.Reader, required []string) (io.Reader, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("ingestion: reading input: %w", err)
	}

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return bytes.NewReader(data), nil
	}

	var records []map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &records); err != nil {
		return nil, fmt.Errorf("ingestion: invalid JSON input: %w", err)
	}

	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	if err := writer.Write(required); err != nil {
		return nil, fmt.Errorf("ingestion: writing normalized header: %w", err)
	}
	for _, record := range records {
		row := make([]string, len(required))
		for i, name := range required {
			value, ok := record[name]
			if !ok {
				continue
			}
			var stringValue string
			if err := json.Unmarshal(value, &stringValue); err == nil {
				row[i] = stringValue
			} else {
				row[i] = string(value)
			}
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("ingestion: writing normalized record: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("ingestion: flushing normalized input: %w", err)
	}
	return bytes.NewReader(output.Bytes()), nil
}
