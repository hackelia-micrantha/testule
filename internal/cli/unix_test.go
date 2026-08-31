package cli

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"syscall"
	"testing"
)

type brokenPipeWriter struct{}

func (brokenPipeWriter) Write([]byte) (int, error) {
	return 0, syscall.EPIPE
}

func TestRunValidateReadsPlanFromStdin(t *testing.T) {
	data, err := os.ReadFile("../../testdata/valid/minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exit := RunWithInput([]string{"validate", "--format=json", "-"}, bytes.NewReader(data), &stdout, &stderr)
	if exit != ExitOK {
		t.Fatalf("expected exit %d, got %d; stderr=%q", ExitOK, exit, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"valid":true`) || !strings.Contains(stdout.String(), `"source":"-"`) {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunFingerprintReadsPlanFromStdin(t *testing.T) {
	data, err := os.ReadFile("../../testdata/valid/minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exit := RunWithInput([]string{"fingerprint", "-"}, bytes.NewReader(data), &stdout, &stderr)
	if exit != ExitOK {
		t.Fatalf("expected exit %d, got %d; stderr=%q", ExitOK, exit, stderr.String())
	}
	if !strings.HasPrefix(strings.TrimSpace(stdout.String()), "sha256:") {
		t.Fatalf("unexpected fingerprint: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunGapsReadsEvidenceFromStdin(t *testing.T) {
	planPath := "../../testdata/valid/gap-plan.yaml"
	var fingerprintOut, stderr bytes.Buffer
	if exit := Run([]string{"fingerprint", planPath}, &fingerprintOut, &stderr); exit != ExitOK {
		t.Fatalf("fingerprint exit=%d stderr=%q", exit, stderr.String())
	}
	fingerprint := strings.TrimSpace(fingerprintOut.String())

	evidenceData := fmt.Sprintf(`apiVersion: testule.dev/v1alpha1
kind: Evidence
metadata:
  name: stdin-evidence
plan:
  name: parser-gap
  fingerprint: %s
subject:
  component: parser
  revision: rev-stdin
environment:
  id: stdin-test
provenance:
  producer: stdin-test
  runId: run-1
observations:
  - id: all-required
    status: passed
    coverage:
      levels: [unit, integration]
      behaviors: [negative]
      generation: [fuzz]
`, fingerprint)

	var stdout bytes.Buffer
	stderr.Reset()
	exit := RunWithInput(
		[]string{"gaps", "--format=json", "--subject-revision=rev-stdin", planPath, "-"},
		strings.NewReader(evidenceData),
		&stdout,
		&stderr,
	)
	if exit != ExitOK {
		t.Fatalf("gaps exit=%d stderr=%q stdout=%q", exit, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"complete":true`) {
		t.Fatalf("unexpected report: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunGapsRejectsAmbiguousStdinBinding(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exit := RunWithInput(
		[]string{"gaps", "--subject-revision=rev-1", "-", "-"},
		strings.NewReader(""),
		&stdout,
		&stderr,
	)
	if exit != ExitUsage {
		t.Fatalf("expected exit %d, got %d", ExitUsage, exit)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunBrokenPipeDoesNotBecomeInternalFailure(t *testing.T) {
	data, err := os.ReadFile("../../testdata/valid/minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	exit := RunWithInput(
		[]string{"validate", "--format=json", "-"},
		bytes.NewReader(data),
		brokenPipeWriter{},
		&stderr,
	)
	if exit != ExitOK {
		t.Fatalf("expected broken pipe to terminate successfully, got exit %d", exit)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no broken-pipe diagnostic, got %q", stderr.String())
	}
}

func TestRunInvalidJSONFromStdinKeepsDiagnosticsOutOfStderr(t *testing.T) {
	data, err := os.ReadFile("../../testdata/invalid/invalid-disposition.yaml")
	if err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	exit := RunWithInput([]string{"validate", "--format=json", "-"}, bytes.NewReader(data), &stdout, &stderr)
	if exit != ExitInvalidPlan {
		t.Fatalf("expected exit %d, got %d", ExitInvalidPlan, exit)
	}
	if !strings.Contains(stdout.String(), `"valid":false`) || !strings.Contains(stdout.String(), `"invalid_value"`) {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr in JSON mode, got %q", stderr.String())
	}
}
