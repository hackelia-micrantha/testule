package adapter_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	adaptercontract "github.com/hackelia-micrantha/testule/internal/adapter"
	"github.com/hackelia-micrantha/testule/internal/adapter/junit"
	pythonadapter "github.com/hackelia-micrantha/testule/internal/adapter/python"
	"github.com/hackelia-micrantha/testule/internal/evidence"
	"github.com/hackelia-micrantha/testule/internal/gap"
	"github.com/hackelia-micrantha/testule/internal/plan"
)

func TestLanguageNeutralUnitRequirementAcrossAdapterShapes(t *testing.T) {
	p := unitPlan()
	fingerprint, err := plan.Fingerprint(p)
	if err != nil {
		t.Fatal(err)
	}
	binding := adaptercontract.PlanBinding{
		APIVersion: p.APIVersion, Name: p.Metadata.Name, Fingerprint: fingerprint, Component: p.Subject.Component,
	}

	goRecord := normalizedGoFixture(binding, "rev-1")
	assertSatisfied(t, p, goRecord, "rev-1")

	junitResult := (junit.Adapter{}).Invoke(context.Background(), adaptercontract.Invocation{
		Capability: adaptercontract.CapabilityEvidenceImport,
		Plan: binding, SubjectRevision: "rev-1", EnvironmentID: "ci-junit", RunID: "junit-conformance",
		Coverage: adaptercontract.Coverage{Level: "unit", Generation: "example"},
		Input: []byte(`<testsuite><testcase classname="fixture.Sample" name="test_ok"/></testsuite>`),
	})
	if junitResult.Status != adaptercontract.StatusCompleted || junitResult.Evidence == nil {
		t.Fatalf("JUnit import failed: %#v", junitResult)
	}
	assertSatisfied(t, p, junitResult.Evidence, "rev-1")

	python := pythonadapter.Adapter{}
	probe := python.Probe(context.Background())
	if len(probe.Availability) == 1 && probe.Availability[0].Available {
		workspace := t.TempDir()
		if err := os.WriteFile(filepath.Join(workspace, "sample_test.py"), []byte("import unittest\nclass Sample(unittest.TestCase):\n    def test_ok(self):\n        self.assertTrue(True)\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		pythonResult := python.Invoke(context.Background(), adaptercontract.Invocation{
			Capability: adaptercontract.CapabilityTestExecute,
			Plan: binding, SubjectRevision: "rev-1", EnvironmentID: "ci-python", RunID: "python-conformance",
			TargetID: "sample_test.Sample.test_ok",
			Coverage: adaptercontract.Coverage{Level: "unit", Generation: "example"},
			AdapterOptions: map[string]string{"python.workspace": workspace},
		})
		if pythonResult.Status != adaptercontract.StatusCompleted || pythonResult.Evidence == nil {
			t.Fatalf("Python execution failed: %#v", pythonResult)
		}
		assertSatisfied(t, p, pythonResult.Evidence, "rev-1")
	}
}

func TestImportedFailureFailsSameRequirementWithoutExecution(t *testing.T) {
	p := unitPlan()
	fingerprint, err := plan.Fingerprint(p)
	if err != nil {
		t.Fatal(err)
	}
	result := (junit.Adapter{}).Invoke(context.Background(), adaptercontract.Invocation{
		Capability: adaptercontract.CapabilityEvidenceImport,
		Plan: adaptercontract.PlanBinding{APIVersion: p.APIVersion, Name: p.Metadata.Name, Fingerprint: fingerprint, Component: p.Subject.Component},
		SubjectRevision: "rev-fail", EnvironmentID: "ci-junit", RunID: "junit-fail",
		Coverage: adaptercontract.Coverage{Level: "unit", Generation: "example"},
		Input: []byte(`<testsuite><testcase classname="fixture.Sample" name="test_bad"><failure/></testcase></testsuite>`),
	})
	if result.Status != adaptercontract.StatusCompleted || result.Evidence == nil || result.Evidence.Execution != nil {
		t.Fatalf("importer must complete without fabricating execution: %#v", result)
	}
	report, diagnostics := gap.Evaluate(p, []*evidence.Evidence{result.Evidence}, "rev-fail")
	if len(diagnostics) != 0 {
		t.Fatalf("gap diagnostics: %#v", diagnostics)
	}
	if report.Complete || report.Summary.Failed != 1 {
		t.Fatalf("failed imported observation did not fail requirement: %#v", report)
	}
}

func unitPlan() *plan.TestPlan {
	required := "required"
	return &plan.TestPlan{
		APIVersion: "testule.dev/v1alpha1",
		Kind: "TestPlan",
		Metadata: &plan.Metadata{Name: "portable-unit"},
		Subject: &plan.Subject{Component: "fixture"},
		Requirements: &plan.Requirements{Levels: &plan.Levels{Unit: &required}},
	}
}

func normalizedGoFixture(binding adaptercontract.PlanBinding, revision string) *evidence.Evidence {
	return &evidence.Evidence{
		APIVersion: binding.APIVersion, Kind: evidence.Kind,
		Metadata: &evidence.Metadata{Name: "go-conformance"},
		Plan: &evidence.PlanReference{Name: binding.Name, Fingerprint: binding.Fingerprint},
		Subject: &evidence.Subject{Component: binding.Component, Revision: revision},
		Environment: &evidence.Environment{ID: "ci-go"},
		Provenance: &evidence.Provenance{Producer: "go-native/v1alpha1", RunID: "go-conformance"},
		Execution: &evidence.Execution{Adapter: "go-native/v1alpha1", Operation: "test", Tool: "go", ToolVersion: "fixture", Scope: "./sample", Target: "TestPass", Command: []string{"go", "test"}},
		Observations: []evidence.Observation{{ID: "go-test:TestPass", Status: "passed", Coverage: evidence.Coverage{Levels: []string{"unit"}, Generation: []string{"example"}}}},
	}
}

func assertSatisfied(t *testing.T, p *plan.TestPlan, record *evidence.Evidence, revision string) {
	t.Helper()
	if diagnostics := evidence.Validate(record); len(diagnostics) != 0 {
		t.Fatalf("invalid normalized evidence: %#v", diagnostics)
	}
	report, diagnostics := gap.Evaluate(p, []*evidence.Evidence{record}, revision)
	if len(diagnostics) != 0 {
		t.Fatalf("gap diagnostics: %#v", diagnostics)
	}
	if !report.Complete || report.Summary.Satisfied != 1 {
		t.Fatalf("unit requirement was not satisfied: %#v", report)
	}
}

var _ = strings.Builder{}
