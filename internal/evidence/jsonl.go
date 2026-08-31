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
// Each non-empty physical line must contain exactly one strict JSON object.
// A complete final record does not require a trailing newline.
func DecodeJSONL(r io.Reader) ([]*Evidence, []plan.Diagnostic) {
	limited := io.LimitReader(r, MaxEvidenceStreamBytes+1)
	scanner := bufio.NewScanner(limited)
	scanner.Buffer(make([]byte, 64*1024), int(MaxEvidenceBytes)+1)

	records := make([]*Evidence, 0)
	line := 0
	for scanner.Scan() {
		line++
		data := bytes.TrimSpace(scanner.Bytes())
		if len(data) == 0 {
			return nil, []plan.Diagnostic{{Code: "invalid_jsonl", Message: fmt.Sprintf("line %d: blank lines are not permitted", line)}}
		}
		if len(records) >= MaxEvidenceRecords {
			return nil, []plan.Diagnostic{{Code: "too_many_records", Message: fmt.Sprintf("evidence stream exceeds maximum of %d records", MaxEvidenceRecords)}}
		}

		var record Evidence
		decoder := json.NewDecoder(bytes.NewReader(data))
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
		if errors.Is(err, bufio.ErrTooLong) || len(records) == 0 {
			return nil, []plan.Diagnostic{{Code: "input_too_large", Message: "evidence JSONL record or stream exceeds the supported size limit"}}
		}
		return nil, []plan.Diagnostic{{Code: "invalid_jsonl", Message: err.Error()}}
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
