package adapter

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type importerFixture struct{}

func (importerFixture) Describe() Descriptor {
	return Descriptor{ProtocolVersion: ProtocolVersion, ID: "junit-xml", Version: "v1alpha1", Capabilities: []Capability{CapabilityEvidenceImport}}
}

func (importerFixture) Probe(context.Context) ProbeResult {
	return ProbeResult{Adapter: "junit-xml", Availability: []Availability{{Capability: CapabilityEvidenceImport, Available: true}}}
}

func (importerFixture) Invoke(context.Context, Invocation) Result {
	return Result{Status: StatusCompleted}
}

func TestImporterDoesNotNeedDiscovery(t *testing.T) {
	var candidate Adapter = importerFixture{}
	if _, ok := candidate.(Discoverer); ok {
		t.Fatal("importer fixture must not fabricate discovery")
	}
}

func TestContractShapesAreJSONSerializableWithoutCommandEnvelope(t *testing.T) {
	invocation := Invocation{
		Capability:      CapabilityTestExecute,
		SubjectRevision: "rev-1",
		EnvironmentID:   "ci",
		RunID:           "run-1",
		TargetID:        "sample.SampleTests.test_ok",
		Coverage:        Coverage{Level: "unit", Generation: "example"},
		AdapterOptions:  map[string]string{"python.module": "sample"},
	}
	encoded, err := json.Marshal(invocation)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	if strings.Contains(text, "command") || strings.Contains(text, "shell") || strings.Contains(text, "executable") {
		t.Fatalf("generic invocation leaked command authority: %s", text)
	}
	var decoded Invocation
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.TargetID != invocation.TargetID {
		t.Fatalf("opaque target changed: got %q want %q", decoded.TargetID, invocation.TargetID)
	}
}

func TestExecutionAndVerificationStatusesAreIndependent(t *testing.T) {
	result := Result{Status: StatusCompleted}
	if result.Status != StatusCompleted {
		t.Fatalf("unexpected terminal status: %q", result.Status)
	}
	// Evidence observation status is intentionally not inferred from the
	// terminal status. A completed adapter operation may normalize failures.
}
