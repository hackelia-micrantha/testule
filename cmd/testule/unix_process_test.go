package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestValidateStdinProcessContract(t *testing.T) {
	binary := buildCLI(t)
	planData, err := os.ReadFile("../../testdata/valid/minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}

	command := exec.Command(binary, "validate", "--format=json", "-")
	command.Stdin = bytes.NewReader(planData)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		t.Fatalf("validate stdin failed: %v; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"valid":true`) || !strings.Contains(stdout.String(), `"source":"-"`) {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestInvalidJSONProcessKeepsDiagnosticsOnStdout(t *testing.T) {
	binary := buildCLI(t)
	planData, err := os.ReadFile("../../testdata/invalid/invalid-disposition.yaml")
	if err != nil {
		t.Fatal(err)
	}

	command := exec.Command(binary, "validate", "--format=json", "-")
	command.Stdin = bytes.NewReader(planData)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err = command.Run()
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 3 {
		t.Fatalf("expected exit 3, got %v; stderr=%q", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"valid":false`) || !strings.Contains(stdout.String(), `"invalid_value"`) {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr in JSON mode, got %q", stderr.String())
	}
}
