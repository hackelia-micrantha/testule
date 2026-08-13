package gap

import (
	"strings"
	"testing"

	"github.com/hackelia-micrantha/testule/internal/evidence"
	"github.com/hackelia-micrantha/testule/internal/plan"
)

func TestEvaluateDistinguishesGapStates(t *testing.T) {
	p := representativePlan()
	fingerprint, err := plan.Fingerprint(p)
	if err != nil {
		t.Fatal(err)
	}

	records := []*evidence.Evidence{
		evidenceRecord("pass", fingerprint, "rev-1", []evidence.Observation{
			{ID: "unit-negative", Status: "passed", Coverage: evidence.Coverage{Levels: []string{"unit"}, Behaviors: []string{"negative"}}},
		}),
		evidenceRecord("unsupported", fingerprint, "rev-1", []evidence.Observation{
			{ID: "integration", Status: "unsupported", Coverage: evidence.Coverage{Levels: []string{"integration"}}},
		}),
		evidenceRecord("skipped", fingerprint, "rev-1", []evidence.Observation{
			{ID: "fuzz", Status: "skipped", Coverage: evidence.Coverage{Generation: []string{"fuzz"}}},
		}),
	}

	report, diagnostics := Evaluate(p, records, "rev-1")
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if report.Complete {
		t.Fatal("expected report to be incomplete")
	}

	assertEntryState(t, report, "level", "unit", StateSatisfied)
	assertEntryState(t, report, "behavior", "negative", StateSatisfied)
	assertEntryState(t, report, "level", "integration", StateUnsupported)
	assertEntryState(t, report, "generation", "fuzz", StateSkipped)
	assertEntryState(t, report, "behavior", "positive", StateMissing)
	assertEntryState(t, report, "level", "endToEnd", StateInapplicable)
}

func TestEvaluateRequiredUnsupportedNeverPasses(t *testing.T) {
	p := representativePlan()
	fingerprint, _ := plan.Fingerprint(p)
	records := []*evidence.Evidence{
		evidenceRecord("unsupported", fingerprint, "rev-1", []evidence.Observation{
			{ID: "unit", Status: "unsupported", Coverage: evidence.Coverage{Levels: []string{"unit"}}},
		}),
	}

	report, diagnostics := Evaluate(p, records, "rev-1")
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if report.Complete {
		t.Fatal("required unsupported capability must not produce a complete report")
	}
	assertEntryState(t, report, "level", "unit", StateUnsupported)
}

func TestEvaluateFailureTakesPrecedenceOverPass(t *testing.T) {
	p := &plan.TestPlan{
		APIVersion: plan.APIVersion,
		Kind:       plan.Kind,
		Metadata:   &plan.Metadata{Name: "parser"},
		Subject:    &plan.Subject{Component: "parser"},
		Requirements: &plan.Requirements{
			Levels: &plan.Levels{Unit: disposition("optional")},
		},
	}
	fingerprint, _ := plan.Fingerprint(p)
	records := []*evidence.Evidence{
		evidenceRecord("pass", fingerprint, "rev-1", []evidence.Observation{
			{ID: "unit-pass", Status: "passed", Coverage: evidence.Coverage{Levels: []string{"unit"}}},
		}),
		evidenceRecord("fail", fingerprint, "rev-1", []evidence.Observation{
			{ID: "unit-fail", Status: "failed", Coverage: evidence.Coverage{Levels: []string{"unit"}}},
		}),
	}

	report, diagnostics := Evaluate(p, records, "rev-1")
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	if report.Complete {
		t.Fatal("observed failure must block completeness even for an optional requirement")
	}
	assertEntryState(t, report, "level", "unit", StateFailed)
}

func TestEvaluateRejectsMismatchedAndDuplicateEvidence(t *testing.T) {
	p := representativePlan()
	fingerprint, _ := plan.Fingerprint(p)
	first := evidenceRecord("same", fingerprint, "stale-revision", []evidence.Observation{
		{ID: "unit", Status: "passed", Coverage: evidence.Coverage{Levels: []string{"unit"}}},
	})
	second := evidenceRecord("same", "sha256:"+strings.Repeat("b", 64), "rev-1", []evidence.Observation{
		{ID: "integration", Status: "passed", Coverage: evidence.Coverage{Levels: []string{"integration"}}},
	})

	_, diagnostics := Evaluate(p, []*evidence.Evidence{first, second}, "rev-1")
	assertDiagnosticCode(t, diagnostics, "duplicate_evidence")
	assertDiagnosticCode(t, diagnostics, "evidence_plan_fingerprint_mismatch")
	assertDiagnosticCode(t, diagnostics, "evidence_subject_revision_mismatch")
}

func representativePlan() *plan.TestPlan {
	return &plan.TestPlan{
		APIVersion: plan.APIVersion,
		Kind:       plan.Kind,
		Metadata:   &plan.Metadata{Name: "parser"},
		Subject:    &plan.Subject{Component: "parser"},
		Requirements: &plan.Requirements{
			Levels: &plan.Levels{
				Unit:        disposition("required"),
				Integration: disposition("required"),
			},
			Behaviors: &plan.Behaviors{
				Positive: disposition("optional"),
				Negative: disposition("required"),
			},
			Generation: &plan.Generation{
				Fuzz: disposition("required"),
			},
			Inapplicable: []plan.Inapplicable{{
				Dimension: "level",
				Value:     "endToEnd",
				Rationale: "No external execution path exists in this slice.",
			}},
		},
	}
}

func evidenceRecord(name, fingerprint, revision string, observations []evidence.Observation) *evidence.Evidence {
	return &evidence.Evidence{
		APIVersion:  plan.APIVersion,
		Kind:        evidence.Kind,
		Metadata:    &evidence.Metadata{Name: name},
		Plan:        &evidence.PlanReference{Name: "parser", Fingerprint: fingerprint},
		Subject:     &evidence.Subject{Component: "parser", Revision: revision},
		Environment: &evidence.Environment{ID: "linux-amd64"},
		Provenance:  &evidence.Provenance{Producer: "fixture", RunID: "run-" + name},
		Observations: observations,
	}
}

func disposition(value string) *string {
	return &value
}

func assertEntryState(t *testing.T, report Report, dimension, value string, expected State) {
	t.Helper()
	for _, entry := range report.Entries {
		if entry.Dimension == dimension && entry.Value == value {
			if entry.State != expected {
				t.Fatalf("%s.%s: expected %s, got %s", dimension, value, expected, entry.State)
			}
			return
		}
	}
	t.Fatalf("missing entry %s.%s", dimension, value)
}

func assertDiagnosticCode(t *testing.T, diagnostics []plan.Diagnostic, code string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return
		}
	}
	t.Fatalf("expected diagnostic %q in %#v", code, diagnostics)
}
