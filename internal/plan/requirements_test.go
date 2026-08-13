package plan

import "testing"

func TestValidateAcceptsGenerationAndInapplicable(t *testing.T) {
	p := &TestPlan{
		APIVersion: APIVersion,
		Kind:       Kind,
		Metadata:   &Metadata{Name: "parser"},
		Subject:    &Subject{Component: "parser"},
		Requirements: &Requirements{
			Levels:     &Levels{Unit: stringPtr("required")},
			Generation: &Generation{Fuzz: stringPtr("required")},
			Inapplicable: []Inapplicable{{
				Dimension: "level",
				Value:     "endToEnd",
				Rationale: "No externally deployed system exists.",
			}},
		},
	}

	if diagnostics := Validate(p); len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
}

func TestValidateRejectsContradictoryInapplicableRequirement(t *testing.T) {
	p := &TestPlan{
		APIVersion: APIVersion,
		Kind:       Kind,
		Metadata:   &Metadata{Name: "parser"},
		Subject:    &Subject{Component: "parser"},
		Requirements: &Requirements{
			Generation: &Generation{Fuzz: stringPtr("required")},
			Inapplicable: []Inapplicable{{
				Dimension: "generation",
				Value:     "fuzz",
				Rationale: "Contradictory by construction.",
			}},
		},
	}

	diagnostics := Validate(p)
	if !hasDiagnosticCode(diagnostics, "contradiction") {
		t.Fatalf("expected contradiction diagnostic, got %#v", diagnostics)
	}
}

func TestDeclaredRequirementsAreDeterministic(t *testing.T) {
	p := &TestPlan{
		Requirements: &Requirements{
			Levels:     &Levels{Unit: stringPtr("required")},
			Behaviors:  &Behaviors{Negative: stringPtr("required")},
			Generation: &Generation{Fuzz: stringPtr("optional")},
			Inapplicable: []Inapplicable{{
				Dimension: "level",
				Value:     "endToEnd",
				Rationale: "not applicable",
			}},
		},
	}

	got := DeclaredRequirements(p)
	if len(got) != 4 {
		t.Fatalf("expected 4 requirements, got %d", len(got))
	}
	for i := 1; i < len(got); i++ {
		prev := got[i-1].Dimension + "." + got[i-1].Value
		current := got[i].Dimension + "." + got[i].Value
		if prev > current {
			t.Fatalf("requirements are not sorted: %#v", got)
		}
	}
}

func stringPtr(value string) *string {
	return &value
}

func hasDiagnosticCode(diagnostics []Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
