package python

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	adaptercontract "github.com/hackelia-micrantha/testule/internal/adapter"
)

func testInvocation(workspace, target string) adaptercontract.Invocation {
	return adaptercontract.Invocation{
		Capability: adaptercontract.CapabilityTestExecute,
		Plan: adaptercontract.PlanBinding{
			APIVersion:  "testule.dev/v1alpha1",
			Name:        "portable-unit",
			Fingerprint: "sha256:" + strings.Repeat("a", 64),
			Component:   "parser",
		},
		SubjectRevision: "rev-python",
		EnvironmentID:   "ci-python",
		RunID:           "python-1",
		TargetID:        target,
		Coverage:        adaptercontract.Coverage{Level: "unit", Generation: "example"},
		AdapterOptions:  map[string]string{"python.workspace": workspace},
	}
}

func writeFixture(t *testing.T, body string) string {
	t.Helper()
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "sample_test.py"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return workspace
}

func requirePython(t *testing.T) Adapter {
	t.Helper()
	adapter := Adapter{}
	probe := adapter.Probe(context.Background())
	if len(probe.Availability) != 1 || !probe.Availability[0].Available {
		t.Skip("python3 unavailable in test environment")
	}
	return adapter
}

func TestPythonExecutionCompletedWithPassedObservation(t *testing.T) {
	adapter := requirePython(t)
	workspace := writeFixture(t, "import unittest\nclass Sample(unittest.TestCase):\n    def test_ok(self):\n        self.assertEqual(2 + 2, 4)\n")
	result := adapter.Invoke(context.Background(), testInvocation(workspace, "sample_test.Sample.test_ok"))
	if result.Status != adaptercontract.StatusCompleted {
		t.Fatalf("execution did not complete: %#v", result)
	}
	if result.Evidence == nil || result.Evidence.Observations[0].Status != "passed" {
		t.Fatalf("unexpected evidence: %#v", result.Evidence)
	}
	if result.Evidence.Execution.Target != "sample_test.Sample.test_ok" {
		t.Fatalf("opaque target changed: %q", result.Evidence.Execution.Target)
	}
}

func TestPythonExecutionCompletedWithFailedObservation(t *testing.T) {
	adapter := requirePython(t)
	workspace := writeFixture(t, "import unittest\nclass Sample(unittest.TestCase):\n    def test_bad(self):\n        self.fail('expected failure')\n")
	result := adapter.Invoke(context.Background(), testInvocation(workspace, "sample_test.Sample.test_bad"))
	if result.Status != adaptercontract.StatusCompleted {
		t.Fatalf("execution did not complete: %#v", result)
	}
	if result.Evidence == nil || result.Evidence.Observations[0].Status != "failed" {
		t.Fatalf("terminal and observation status collapsed: %#v", result)
	}
}

func TestMissingPythonIsUnsupported(t *testing.T) {
	adapter := Adapter{Executable: "testule-python-definitely-missing"}
	probe := adapter.Probe(context.Background())
	if len(probe.Availability) != 1 || probe.Availability[0].Available {
		t.Fatalf("unexpected probe: %#v", probe)
	}
	workspace := t.TempDir()
	result := adapter.Invoke(context.Background(), testInvocation(workspace, "sample_test.Sample.test_ok"))
	if result.Status != adaptercontract.StatusUnsupported {
		t.Fatalf("status=%q diagnostics=%v", result.Status, result.Diagnostics)
	}
}

func TestPythonRejectsUnknownOptionsAndControlCharacters(t *testing.T) {
	request := testInvocation(t.TempDir(), "sample_test.Sample.test_ok")
	request.AdapterOptions["command"] = "rm -rf /"
	if result := (Adapter{}).Invoke(context.Background(), request); result.Status != adaptercontract.StatusInfrastructureFailed {
		t.Fatalf("unknown adapter option accepted: %#v", result)
	}

	request = testInvocation(t.TempDir(), "sample_test.Sample.test_ok\nother")
	if result := (Adapter{}).Invoke(context.Background(), request); result.Status != adaptercontract.StatusInfrastructureFailed {
		t.Fatalf("control character accepted: %#v", result)
	}
}
