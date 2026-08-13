package evidence

import (
	"strings"
	"testing"
)

func TestValidateExecutionArtifactsAcceptsBoundedMetadata(t *testing.T) {
	record := &Evidence{
		Execution: &Execution{
			Adapter: "go-native/v1alpha1", Operation: "test", Tool: "go", ToolVersion: "go1.26.5",
			Package: "./sample", Target: "TestPass", Command: []string{"go", "test", "./sample"}, ExitCode: 0, DurationMillis: 12,
		},
		Artifacts: []Artifact{{Name: "stdout.log", Role: "stdout", Path: "results/stdout.log", SHA256: "sha256:" + strings.Repeat("a", 64), MediaType: "text/plain"}},
	}
	if diagnostics := ValidateExecutionArtifacts(record); len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
}

func TestValidateExecutionArtifactsRejectsTraversalAndBadDigest(t *testing.T) {
	record := &Evidence{
		Execution: &Execution{Adapter: "go-native/v1alpha1", Operation: "fuzz", Tool: "go", ToolVersion: "go1.26.5", Command: []string{"go", "test"}},
		Artifacts: []Artifact{{Name: "reproducer", Role: "fuzz-reproducer", Path: "../escape", SHA256: "bad"}},
	}
	diagnostics := ValidateExecutionArtifacts(record)
	assertCode := func(code string) {
		t.Helper()
		for _, diagnostic := range diagnostics {
			if diagnostic.Code == code {
				return
			}
		}
		t.Fatalf("expected %s in %#v", code, diagnostics)
	}
	assertCode("invalid_value")
}

func TestValidateExecutionArtifactsRejectsDuplicateArtifactIdentity(t *testing.T) {
	digest := "sha256:" + strings.Repeat("b", 64)
	record := &Evidence{Artifacts: []Artifact{
		{Name: "same", Role: "stdout", Path: "results/a", SHA256: digest},
		{Name: "same", Role: "stderr", Path: "results/a", SHA256: digest},
	}}
	diagnostics := ValidateExecutionArtifacts(record)
	count := 0
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == "duplicate" {
			count++
		}
	}
	if count < 2 {
		t.Fatalf("expected duplicate name and path diagnostics, got %#v", diagnostics)
	}
}
