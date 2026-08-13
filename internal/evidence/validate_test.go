package evidence

import (
	"strings"
	"testing"

	"github.com/hackelia-micrantha/testule/internal/plan"
)

func TestDecodeValidEvidence(t *testing.T) {
	data := []byte(`apiVersion: testule.dev/v1alpha1
kind: Evidence
metadata:
  name: unit-run
plan:
  name: parser
  fingerprint: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
subject:
  component: parser
  revision: rev-1
environment:
  id: linux-amd64
provenance:
  producer: go-test
  runId: run-1
  references:
    - artifact://junit.xml
observations:
  - id: unit-negative
    status: passed
    coverage:
      levels: [unit]
      behaviors: [negative]
      visibility: [whiteBox]
      qualityAttributes: [functional]
`)
	record, diagnostics := Decode(data)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if record.Metadata.Name != "unit-run" || len(record.Observations) != 1 {
		t.Fatalf("unexpected record: %#v", record)
	}
}

func TestDecodeRejectsUnknownField(t *testing.T) {
	data := []byte(`apiVersion: testule.dev/v1alpha1
kind: Evidence
metadata:
  name: bad
unexpected: true
`)
	_, diagnostics := Decode(data)
	assertEvidenceDiagnostic(t, diagnostics, "unknown_field")
}

func TestValidateRejectsDuplicateAndIncompleteObservations(t *testing.T) {
	record := validRecord()
	record.Observations = []Observation{
		{ID: "same", Status: "passed", Coverage: Coverage{Levels: []string{"unit"}}},
		{ID: "same", Status: "failed", Coverage: Coverage{}},
	}
	diagnostics := Validate(record)
	assertEvidenceDiagnostic(t, diagnostics, "duplicate")
	assertEvidenceDiagnostic(t, diagnostics, "required")
}

func TestValidateRejectsInvalidFingerprintAndOversizedReference(t *testing.T) {
	record := validRecord()
	record.Plan.Fingerprint = "sha256:ABC"
	record.Provenance.References = []string{strings.Repeat("x", 2049)}
	diagnostics := Validate(record)
	assertEvidenceDiagnostic(t, diagnostics, "invalid_value")
	assertEvidenceDiagnostic(t, diagnostics, "too_long")
}

func validRecord() *Evidence {
	return &Evidence{
		APIVersion: plan.APIVersion,
		Kind:       Kind,
		Metadata:   &Metadata{Name: "unit-run"},
		Plan: &PlanReference{
			Name:        "parser",
			Fingerprint: "sha256:" + strings.Repeat("a", 64),
		},
		Subject:     &Subject{Component: "parser", Revision: "rev-1"},
		Environment: &Environment{ID: "linux-amd64"},
		Provenance:  &Provenance{Producer: "go-test", RunID: "run-1"},
		Observations: []Observation{{
			ID:       "unit",
			Status:   "passed",
			Coverage: Coverage{Levels: []string{"unit"}},
		}},
	}
}

func assertEvidenceDiagnostic(t *testing.T, diagnostics []plan.Diagnostic, code string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return
		}
	}
	t.Fatalf("expected diagnostic %q in %#v", code, diagnostics)
}
