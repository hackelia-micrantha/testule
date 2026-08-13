package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGoAdapterEndToEndEvidenceAndGaps(t *testing.T) {
	binary := buildTestuleBinary(t)
	workspace := copyGoAdapterFixture(t)
	planPath := filepath.Clean(filepath.Join("..", "..", "testdata", "valid", "gap-plan.yaml"))

	unitOutput, err := exec.Command(binary,
		"go", "test",
		"--plan", planPath,
		"--subject-revision", "rev-e2e",
		"--workspace", workspace,
		"--package", "./sample",
		"--target", "TestPass",
		"--level", "unit",
		"--behavior", "positive",
		"--environment", "ci-linux",
		"--run-id", "e2e-unit",
	).CombinedOutput()
	if err != nil {
		t.Fatalf("go test adapter failed: %v\n%s", err, unitOutput)
	}
	unitEvidence := evidencePathFromOutput(t, string(unitOutput))

	fuzzOutput, err := exec.Command(binary,
		"go", "fuzz",
		"--plan", planPath,
		"--subject-revision", "rev-e2e",
		"--workspace", workspace,
		"--package", "./sample",
		"--target", "FuzzSafe",
		"--level", "unit",
		"--behavior", "negative",
		"--environment", "ci-linux",
		"--run-id", "e2e-fuzz",
		"--fuzztime", "100ms",
		"--timeout", "15s",
	).CombinedOutput()
	if err != nil {
		t.Fatalf("go fuzz adapter failed: %v\n%s", err, fuzzOutput)
	}
	fuzzEvidence := evidencePathFromOutput(t, string(fuzzOutput))

	gapOutput, err := exec.Command(binary,
		"gaps", "--subject-revision", "rev-e2e", planPath, unitEvidence, fuzzEvidence,
	).CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 5 {
		t.Fatalf("expected gap exit 5, got %v\n%s", err, gapOutput)
	}
	text := string(gapOutput)
	for _, expected := range []string{
		"level.unit [required] satisfied",
		"behavior.negative [required] satisfied",
		"generation.fuzz [required] satisfied",
		"level.integration [required] missing",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("gap output missing %q:\n%s", expected, text)
		}
	}
}

func TestGoAdapterFailingTestReturnsFailureEvidence(t *testing.T) {
	binary := buildTestuleBinary(t)
	workspace := copyGoAdapterFixture(t)
	planPath := filepath.Clean(filepath.Join("..", "..", "testdata", "valid", "gap-plan.yaml"))

	output, err := exec.Command(binary,
		"go", "test",
		"--plan", planPath,
		"--subject-revision", "rev-fail",
		"--workspace", workspace,
		"--package", "./sample",
		"--target", "TestFail",
		"--level", "unit",
		"--behavior", "negative",
		"--environment", "ci-linux",
		"--run-id", "e2e-fail",
	).CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 6 {
		t.Fatalf("expected adapter failure exit 6, got %v\n%s", err, output)
	}
	evidencePath := evidencePathFromOutput(t, string(output))
	data, readErr := os.ReadFile(evidencePath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(data), `"status": "failed"`) || !strings.Contains(string(data), `"adapter": "go-native/v1alpha1"`) {
		t.Fatalf("failure evidence missing normalized execution metadata:\n%s", data)
	}
}

func copyGoAdapterFixture(t *testing.T) string {
	t.Helper()
	source := filepath.Clean(filepath.Join("..", "..", "testdata", "go-adapter"))
	workspace := t.TempDir()
	copyProcessFixtureFile(t, filepath.Join(source, "go.mod"), filepath.Join(workspace, "go.mod"))
	if err := os.MkdirAll(filepath.Join(workspace, "sample"), 0o750); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"sample.go", "sample_test.go"} {
		copyProcessFixtureFile(t, filepath.Join(source, "sample", name), filepath.Join(workspace, "sample", name))
	}
	return workspace
}

func copyProcessFixtureFile(t *testing.T, source, destination string) {
	t.Helper()
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, data, 0o640); err != nil {
		t.Fatal(err)
	}
}

func buildTestuleBinary(t *testing.T) string {
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

func evidencePathFromOutput(t *testing.T, output string) string {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "evidence: ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "evidence: "))
		}
	}
	t.Fatalf("missing evidence path in output: %q", output)
	return ""
}
