package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidateProcessContract(t *testing.T) {
	binary := buildCLI(t)

	valid := exec.Command(binary, "validate", "../../testdata/valid/minimal.yaml")
	output, err := valid.CombinedOutput()
	if err != nil {
		t.Fatalf("valid plan failed: %v\n%s", err, output)
	}
	if !strings.HasPrefix(string(output), "valid: ") {
		t.Fatalf("unexpected valid output: %q", output)
	}

	invalid := exec.Command(binary, "validate", "../../testdata/invalid/missing-name.yaml")
	output, err = invalid.CombinedOutput()
	var exitErr *exec.ExitError
	if !strings.Contains(string(output), "metadata.name") {
		t.Fatalf("unexpected invalid output: %q", output)
	}
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 3 {
		t.Fatalf("expected process exit 3, got %v", err)
	}
}

func TestGapProcessContract(t *testing.T) {
	binary := buildCLI(t)
	planPath := "../../testdata/valid/gap-plan.yaml"

	fingerprintCommand := exec.Command(binary, "fingerprint", planPath)
	output, err := fingerprintCommand.CombinedOutput()
	if err != nil {
		t.Fatalf("fingerprint failed: %v\n%s", err, output)
	}
	fingerprint := strings.TrimSpace(string(output))

	evidencePath := filepath.Join(t.TempDir(), "evidence.yaml")
	evidenceData := fmt.Sprintf(`apiVersion: testule.dev/v1alpha1
kind: Evidence
metadata:
  name: process-run
plan:
  name: parser-gap
  fingerprint: %s
subject:
  component: parser
  revision: rev-process
environment:
  id: process-test
provenance:
  producer: process-test
  runId: run-1
observations:
  - id: all-required
    status: passed
    coverage:
      levels: [unit, integration]
      behaviors: [negative]
      generation: [fuzz]
`, fingerprint)
	if err := os.WriteFile(evidencePath, []byte(evidenceData), 0o600); err != nil {
		t.Fatal(err)
	}

	gaps := exec.Command(binary, "gaps", "--format=json", "--subject-revision=rev-process", planPath, evidencePath)
	output, err = gaps.CombinedOutput()
	if err != nil {
		t.Fatalf("gaps failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), `"complete":true`) || !strings.Contains(string(output), `"state":"satisfied"`) {
		t.Fatalf("unexpected gap output: %q", output)
	}
}

func buildCLI(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "testule")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}

	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	return binary
}
