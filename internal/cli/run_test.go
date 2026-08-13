package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunValidText(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := Run([]string{"validate", "../../testdata/valid/minimal.yaml"}, &stdout, &stderr)
	if exit != ExitOK {
		t.Fatalf("expected exit %d, got %d; stderr=%q", ExitOK, exit, stderr.String())
	}
	if !strings.HasPrefix(stdout.String(), "valid: ") {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunValidJSONUsesEmptyDiagnosticsArray(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := Run([]string{"validate", "--format=json", "../../testdata/valid/minimal.yaml"}, &stdout, &stderr)
	if exit != ExitOK {
		t.Fatalf("expected exit %d, got %d; stderr=%q", ExitOK, exit, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"valid":true`) || !strings.Contains(stdout.String(), `"diagnostics":[]`) {
		t.Fatalf("unexpected JSON: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr in JSON mode, got %q", stderr.String())
	}
}

func TestRunInvalidJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := Run([]string{"validate", "--format=json", "../../testdata/invalid/invalid-disposition.yaml"}, &stdout, &stderr)
	if exit != ExitInvalidPlan {
		t.Fatalf("expected exit %d, got %d", ExitInvalidPlan, exit)
	}
	if !strings.Contains(stdout.String(), `"valid":false`) || !strings.Contains(stdout.String(), `"invalid_value"`) {
		t.Fatalf("unexpected JSON: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr in JSON mode, got %q", stderr.String())
	}
}

func TestRunFingerprintAndCompleteGapReport(t *testing.T) {
	planPath := "../../testdata/valid/gap-plan.yaml"
	var fingerprintOut, stderr bytes.Buffer
	exit := Run([]string{"fingerprint", planPath}, &fingerprintOut, &stderr)
	if exit != ExitOK {
		t.Fatalf("fingerprint exit=%d stderr=%q", exit, stderr.String())
	}
	fingerprint := strings.TrimSpace(fingerprintOut.String())
	if !strings.HasPrefix(fingerprint, "sha256:") {
		t.Fatalf("unexpected fingerprint: %q", fingerprint)
	}

	evidencePath := filepath.Join(t.TempDir(), "evidence.yaml")
	evidenceData := fmt.Sprintf(`apiVersion: testule.dev/v1alpha1
kind: Evidence
metadata:
  name: all-required
plan:
  name: parser-gap
  fingerprint: %s
subject:
  component: parser
  revision: rev-1
environment:
  id: linux-amd64
provenance:
  producer: fixture
  runId: run-1
observations:
  - id: required
    status: passed
    coverage:
      levels: [unit, integration]
      behaviors: [negative]
      generation: [fuzz]
`, fingerprint)
	if err := os.WriteFile(evidencePath, []byte(evidenceData), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	stderr.Reset()
	exit = Run([]string{"gaps", "--format=json", "--subject-revision=rev-1", planPath, evidencePath}, &stdout, &stderr)
	if exit != ExitOK {
		t.Fatalf("gaps exit=%d stderr=%q stdout=%q", exit, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"complete":true`) || !strings.Contains(stdout.String(), `"inapplicable":1`) {
		t.Fatalf("unexpected report: %q", stdout.String())
	}
}

func TestRunGapsReturnsDedicatedExitForRequiredGap(t *testing.T) {
	planPath := "../../testdata/valid/gap-plan.yaml"
	var fingerprintOut, stderr bytes.Buffer
	if exit := Run([]string{"fingerprint", planPath}, &fingerprintOut, &stderr); exit != ExitOK {
		t.Fatalf("fingerprint exit=%d stderr=%q", exit, stderr.String())
	}
	fingerprint := strings.TrimSpace(fingerprintOut.String())

	evidencePath := filepath.Join(t.TempDir(), "evidence.yaml")
	evidenceData := fmt.Sprintf(`apiVersion: testule.dev/v1alpha1
kind: Evidence
metadata:
  name: partial
plan:
  name: parser-gap
  fingerprint: %s
subject:
  component: parser
  revision: rev-1
environment:
  id: linux-amd64
provenance:
  producer: fixture
  runId: run-1
observations:
  - id: unit
    status: passed
    coverage:
      levels: [unit]
`, fingerprint)
	if err := os.WriteFile(evidencePath, []byte(evidenceData), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	stderr.Reset()
	exit := Run([]string{"gaps", "--subject-revision", "rev-1", planPath, evidencePath}, &stdout, &stderr)
	if exit != ExitGaps {
		t.Fatalf("expected exit %d, got %d; stderr=%q", ExitGaps, exit, stderr.String())
	}
	if !strings.Contains(stdout.String(), "level.integration [required] missing") {
		t.Fatalf("unexpected report: %q", stdout.String())
	}
}

func TestRunUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := Run([]string{"validate", "--format=xml", "plan.yaml"}, &stdout, &stderr)
	if exit != ExitUsage {
		t.Fatalf("expected exit %d, got %d", ExitUsage, exit)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunRejectsOversizedFileWithoutUnboundedRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.yaml")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(planMaxBytesForTest() + 1); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exit := Run([]string{"validate", path}, &stdout, &stderr)
	if exit != ExitIO {
		t.Fatalf("expected exit %d, got %d", ExitIO, exit)
	}
	if !strings.Contains(stderr.String(), "maximum supported size") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func planMaxBytesForTest() int64 {
	return 1 << 20
}
