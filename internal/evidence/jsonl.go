package evidence

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/hackelia-micrantha/testule/internal/plan"
)

const (
	MaxEvidenceStreamBytes int64 = 16 << 20
	MaxEvidenceRecords           = 128
)

// DecodeJSONL decodes a bounded stream of normalized Evidence records.
// Each physical line must contain exactly one strict JSON object. A complete
// final record does not require a trailing newline.
func DecodeJSONL(r io.Reader) ([]*Evidence, []plan.Diagnostic) {
	data, err := io.ReadAll(io.LimitReader(r, MaxEvidenceStreamBytes+1))
	if err != nil {
		return nil, []plan.Diagnostic{{Code: "io_error", Message: err.Error()}}
	}
	if int64(len(data)) > MaxEvidenceStreamBytes {
		return nil, []plan.Diagnostic{{Code: "input_too_large", Message: fmt.Sprintf("evidence JSONL stream exceeds maximum size of %d bytes", MaxEvidenceStreamBytes)}}
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 64*1024), int(MaxEvidenceBytes)+1)

	records := make([]*Evidence, 0)
	line := 0
	for scanner.Scan() {
		line++
		raw := bytes.TrimSpace(scanner.Bytes())
		if len(raw) == 0 {
			return nil, []plan.Diagnostic{{Code: "invalid_jsonl", Message: fmt.Sprintf("line %d: blank lines are not permitted", line)}}
		}
		if len(records) >= MaxEvidenceRecords {
			return nil, []plan.Diagnostic{{Code: "too_many_records", Message: fmt.Sprintf("evidence stream exceeds maximum of %d records", MaxEvidenceRecords)}}
		}

		var record Evidence
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&record); err != nil {
			return nil, []plan.Diagnostic{{Code: "invalid_jsonl", Message: fmt.Sprintf("line %d: %v", line, err)}}
		}
		var extra any
		if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
			if err == nil {
				return nil, []plan.Diagnostic{{Code: "invalid_jsonl", Message: fmt.Sprintf("line %d: exactly one JSON object is required", line)}}
			}
			return nil, []plan.Diagnostic{{Code: "invalid_jsonl", Message: fmt.Sprintf("line %d: %v", line, err)}}
		}

		diagnostics := Validate(&record)
		diagnostics = append(diagnostics, ValidateExecutionArtifacts(&record)...)
		sortDiagnostics(diagnostics)
		if len(diagnostics) != 0 {
			for i := range diagnostics {
				diagnostics[i].Message = fmt.Sprintf("line %d: %s", line, diagnostics[i].Message)
			}
			return nil, diagnostics
		}
		records = append(records, &record)
	}
	if err := scanner.Err(); err != nil {
		return nil, []plan.Diagnostic{{Code: "input_too_large", Message: fmt.Sprintf("evidence JSONL record exceeds maximum size of %d bytes", MaxEvidenceBytes)}}
	}
	if len(records) == 0 {
		return nil, []plan.Diagnostic{{Code: "invalid_jsonl", Message: "evidence JSONL stream must contain at least one record"}}
	}
	return records, nil
}

func EncodeJSONL(w io.Writer, record *Evidence) error {
	if record == nil {
		return errors.New("evidence record is nil")
	}
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(record)
}
