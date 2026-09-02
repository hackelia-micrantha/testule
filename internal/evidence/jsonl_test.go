package evidence

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONLRoundTripPreservesIdentityAndProvenance(t *testing.T) {
	record := &Evidence{
		APIVersion:   "testule.dev/v1alpha1",
		Kind:         "Evidence",
		Metadata:     &Metadata{Name: "stream-run"},
		Plan:         &PlanReference{Name: "parser", Fingerprint: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		Subject:      &Subject{Component: "parser", Revision: "rev-stream"},
		Environment:  &Environment{ID: "ci-linux"},
		Provenance:   &Provenance{Producer: "go-native/v1alpha1", RunID: "run-stream", References: []string{"source:fixture"}},
		Observations: []Observation{{ID: "unit", Status: "passed", Coverage: Coverage{Levels: []string{"unit"}}}},
	}

	var stream bytes.Buffer
	if err := EncodeJSONL(&stream, record); err != nil {
		t.Fatal(err)
	}
	records, diagnostics := DecodeJSONL(&stream)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if len(records) != 1 {
		t.Fatalf("expected one record, got %d", len(records))
	}
	got := records[0]
	if got.APIVersion != record.APIVersion || got.Kind != record.Kind || got.Metadata.Name != record.Metadata.Name {
		t.Fatalf("record identity changed: %#v", got)
	}
	if got.Plan.Fingerprint != record.Plan.Fingerprint || got.Subject.Revision != record.Subject.Revision {
		t.Fatalf("plan/subject identity changed: %#v", got)
	}
	if got.Provenance.Producer != record.Provenance.Producer || got.Provenance.RunID != record.Provenance.RunID || got.Provenance.References[0] != record.Provenance.References[0] {
		t.Fatalf("provenance changed: %#v", got.Provenance)
	}
}

func TestDecodeJSONLAcceptsMultipleRecordsAndFinalLineWithoutNewline(t *testing.T) {
	line := `{"apiVersion":"testule.dev/v1alpha1","kind":"Evidence","metadata":{"name":"run"},"plan":{"name":"parser","fingerprint":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"subject":{"component":"parser","revision":"rev"},"environment":{"id":"ci"},"provenance":{"producer":"test","runId":"run"},"observations":[{"id":"unit","status":"passed","coverage":{"levels":["unit"]}}]}`
	records, diagnostics := DecodeJSONL(strings.NewReader(line + "\n" + line))
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if len(records) != 2 {
		t.Fatalf("expected two records, got %d", len(records))
	}
}

func TestDecodeJSONLRejectsBlankLine(t *testing.T) {
	valid := `{"apiVersion":"testule.dev/v1alpha1","kind":"Evidence","metadata":{"name":"run"},"plan":{"name":"parser","fingerprint":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"subject":{"component":"parser","revision":"rev"},"environment":{"id":"ci"},"provenance":{"producer":"test","runId":"run"},"observations":[{"id":"unit","status":"passed","coverage":{"levels":["unit"]}}]}`
	_, diagnostics := DecodeJSONL(strings.NewReader(valid + "\n\n" + valid + "\n"))
	if len(diagnostics) != 1 || diagnostics[0].Code != "invalid_jsonl" || !strings.Contains(diagnostics[0].Message, "line 2") {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
}

func TestDecodeJSONLRejectsMalformedAndUnknownFields(t *testing.T) {
	for name, input := range map[string]string{
		"truncated":     `{"apiVersion":"testule.dev/v1alpha1"`,
		"unknown-field": `{"apiVersion":"testule.dev/v1alpha1","kind":"Evidence","unexpected":true}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, diagnostics := DecodeJSONL(strings.NewReader(input))
			if len(diagnostics) == 0 || diagnostics[0].Code != "invalid_jsonl" {
				t.Fatalf("unexpected diagnostics: %#v", diagnostics)
			}
		})
	}
}

func TestEncodeJSONLIsOneJSONRecordPerLine(t *testing.T) {
	record := &Evidence{
		APIVersion:   "testule.dev/v1alpha1",
		Kind:         "Evidence",
		Metadata:     &Metadata{Name: "run"},
		Plan:         &PlanReference{Name: "parser", Fingerprint: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		Subject:      &Subject{Component: "parser", Revision: "rev"},
		Environment:  &Environment{ID: "ci"},
		Provenance:   &Provenance{Producer: "test", RunID: "run"},
		Observations: []Observation{{ID: "unit", Status: "passed", Coverage: Coverage{Levels: []string{"unit"}}}},
	}
	var output bytes.Buffer
	if err := EncodeJSONL(&output, record); err != nil {
		t.Fatal(err)
	}
	if strings.Count(output.String(), "\n") != 1 || !strings.HasSuffix(output.String(), "\n") {
		t.Fatalf("expected one newline-delimited record: %q", output.String())
	}
	var decoded Evidence
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &decoded); err != nil {
		t.Fatalf("output is not strict JSON: %v", err)
	}
}
