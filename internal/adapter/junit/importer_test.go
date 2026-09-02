package junit

import (
	"context"
	"strings"
	"testing"

	adaptercontract "github.com/hackelia-micrantha/testule/internal/adapter"
)

func invocation(input string) adaptercontract.Invocation {
	return adaptercontract.Invocation{
		Capability: adaptercontract.CapabilityEvidenceImport,
		Plan: adaptercontract.PlanBinding{
			APIVersion:  "testule.dev/v1alpha1",
			Name:        "portable-unit",
			Fingerprint: "sha256:" + strings.Repeat("a", 64),
			Component:   "parser",
		},
		SubjectRevision: "rev-1",
		EnvironmentID:   "ci",
		RunID:           "junit-1",
		Coverage:        adaptercontract.Coverage{Level: "unit", Generation: "example"},
		Input:           []byte(input),
	}
}

func TestImportCompletedCanContainFailedObservation(t *testing.T) {
	result := (Adapter{}).Invoke(context.Background(), invocation(`<testsuite><testcase classname="pkg.Sample" name="test_ok"/><testcase classname="pkg.Sample" name="test_bad"><failure/></testcase></testsuite>`))
	if result.Status != adaptercontract.StatusCompleted {
		t.Fatalf("status=%q diagnostics=%v", result.Status, result.Diagnostics)
	}
	if result.Evidence == nil || len(result.Evidence.Observations) != 2 {
		t.Fatalf("unexpected evidence: %#v", result.Evidence)
	}
	if result.Evidence.Observations[0].Status != "passed" || result.Evidence.Observations[1].Status != "failed" {
		t.Fatalf("unexpected observations: %#v", result.Evidence.Observations)
	}
}

func TestImporterHasNoDiscoveryCapability(t *testing.T) {
	var candidate adaptercontract.Adapter = Adapter{}
	if _, ok := candidate.(adaptercontract.Discoverer); ok {
		t.Fatal("JUnit importer must not fabricate target discovery")
	}
}

func TestImporterRejectsUnsupportedCapability(t *testing.T) {
	request := invocation(`<testsuite><testcase name="ok"/></testsuite>`)
	request.Capability = adaptercontract.CapabilityTestExecute
	result := (Adapter{}).Invoke(context.Background(), request)
	if result.Status != adaptercontract.StatusUnsupported {
		t.Fatalf("status=%q", result.Status)
	}
}

func TestImporterRejectsUnusedTargetAndOptions(t *testing.T) {
	base := `<testsuite><testcase name="ok"/></testsuite>`
	request := invocation(base)
	request.TargetID = "some.target"
	if result := (Adapter{}).Invoke(context.Background(), request); result.Status != adaptercontract.StatusInfrastructureFailed {
		t.Fatalf("unused target was silently accepted: %#v", result)
	}

	request = invocation(base)
	request.AdapterOptions = map[string]string{"command": "rm -rf /"}
	if result := (Adapter{}).Invoke(context.Background(), request); result.Status != adaptercontract.StatusInfrastructureFailed {
		t.Fatalf("unused options were silently accepted: %#v", result)
	}
}

func TestImporterRejectsDTDAndEntityDeclarations(t *testing.T) {
	for _, input := range []string{
		`<!DOCTYPE testsuite SYSTEM "file:///etc/passwd"><testsuite><testcase name="x"/></testsuite>`,
		`<!DOCTYPE testsuite [<!ENTITY xxe SYSTEM "file:///etc/passwd">]><testsuite><testcase name="&xxe;"/></testsuite>`,
	} {
		result := (Adapter{}).Invoke(context.Background(), invocation(input))
		if result.Status != adaptercontract.StatusInfrastructureFailed {
			t.Fatalf("expected fail-closed result for %q: %#v", input, result)
		}
	}
}

func TestImporterTreatsCommandLookingContentAsData(t *testing.T) {
	result := (Adapter{}).Invoke(context.Background(), invocation(`<testsuite><testcase classname="$(touch /tmp/pwned)" name="; rm -rf /"/></testsuite>`))
	if result.Status != adaptercontract.StatusCompleted || result.Evidence == nil {
		t.Fatalf("unexpected result: %#v", result)
	}
	if !strings.Contains(result.Evidence.Observations[0].ID, "touch") {
		t.Fatalf("native label was not preserved as inert data: %q", result.Evidence.Observations[0].ID)
	}
}

func TestImporterRejectsMalformedAndOversizedInput(t *testing.T) {
	for name, input := range map[string]string{
		"malformed": `<testsuite><testcase>`,
		"oversized": strings.Repeat("x", MaxInputBytes+1),
	} {
		t.Run(name, func(t *testing.T) {
			result := (Adapter{}).Invoke(context.Background(), invocation(input))
			if result.Status != adaptercontract.StatusInfrastructureFailed {
				t.Fatalf("unexpected result: %#v", result)
			}
		})
	}
}
