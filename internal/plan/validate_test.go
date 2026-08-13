package plan

import (
	"strings"
	"testing"
)

const validPlan = `apiVersion: testule.dev/v1alpha1
kind: TestPlan
metadata:
  name: parser
subject:
  component: parser
requirements:
  levels:
    unit: required
    integration: required
  behaviors:
    positive: required
    negative: required
    boundary: optional
`

func TestDecodeValidPlan(t *testing.T) {
	p, diagnostics := Decode([]byte(validPlan))
	if p == nil {
		t.Fatal("expected decoded plan")
	}
	if len(diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", diagnostics)
	}
}

func TestDecodeRejectsUnknownField(t *testing.T) {
	input := strings.Replace(validPlan, "  name: parser", "  name: parser\n  unexpected: true", 1)
	_, diagnostics := Decode([]byte(input))
	assertDiagnosticCode(t, diagnostics, "unknown_field")
}

func TestDecodeRejectsMultipleDocuments(t *testing.T) {
	_, diagnostics := Decode([]byte(validPlan + "---\n{}\n"))
	assertDiagnosticCode(t, diagnostics, "multiple_documents")
}

func TestValidateRequiredFieldsAndDisposition(t *testing.T) {
	input := `apiVersion: wrong/v1
kind: Wrong
metadata:
  name: ""
subject:
  component: ""
requirements:
  behaviors:
    negative: sometimes
`
	_, diagnostics := Decode([]byte(input))

	for _, code := range []string{"invalid_api_version", "invalid_kind", "required", "invalid_value"} {
		assertDiagnosticCode(t, diagnostics, code)
	}
}

func TestValidateRequiresAtLeastOneRequirement(t *testing.T) {
	input := `apiVersion: testule.dev/v1alpha1
kind: TestPlan
metadata:
  name: parser
subject:
  component: parser
requirements:
  levels: {}
`
	_, diagnostics := Decode([]byte(input))
	assertDiagnosticCode(t, diagnostics, "no_requirements")
}

func TestValidateNameLengthUsesUnicodeCodePoints(t *testing.T) {
	name := strings.Repeat("é", 129)
	input := strings.Replace(validPlan, "name: parser", "name: "+name, 1)
	_, diagnostics := Decode([]byte(input))
	assertDiagnosticCode(t, diagnostics, "too_long")
}

func TestValidateRejectsControlCharacterInName(t *testing.T) {
	input := strings.Replace(validPlan, "name: parser", `name: "bad\tname"`, 1)
	_, diagnostics := Decode([]byte(input))
	assertDiagnosticCode(t, diagnostics, "control_character")
}

func TestDecodeRejectsMalformedYAML(t *testing.T) {
	_, diagnostics := Decode([]byte("metadata: [\n"))
	assertDiagnosticCode(t, diagnostics, "invalid_yaml")
}

func FuzzDecodeNeverPanics(f *testing.F) {
	f.Add([]byte(validPlan))
	f.Add([]byte("{}\n"))
	f.Add([]byte("metadata: [\n"))

	f.Fuzz(func(t *testing.T, input []byte) {
		if int64(len(input)) > MaxPlanBytes {
			input = input[:MaxPlanBytes]
		}
		_, _ = Decode(input)
	})
}

func TestDecodeRejectsOversizedInput(t *testing.T) {
	_, diagnostics := Decode(make([]byte, MaxPlanBytes+1))
	assertDiagnosticCode(t, diagnostics, "input_too_large")
}

func assertDiagnosticCode(t *testing.T, diagnostics []Diagnostic, code string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return
		}
	}
	t.Fatalf("expected diagnostic code %q, got %#v", code, diagnostics)
}
