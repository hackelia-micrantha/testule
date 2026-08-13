package main

import (
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidateProcessContract(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "testule")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}

	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}

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
