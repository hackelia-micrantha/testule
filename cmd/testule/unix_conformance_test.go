package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestManPageMatchesExecutableUsageInventory(t *testing.T) {
	binary := buildCLI(t)
	manData, err := os.ReadFile("../../man/man1/testule.1")
	if err != nil {
		t.Fatal(err)
	}
	manText := string(manData)

	for _, args := range [][]string{{}, {"go"}} {
		command := exec.Command(binary, args...)
		var stdout, stderr bytes.Buffer
		command.Stdout = &stdout
		command.Stderr = &stderr
		err := command.Run()
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) || exitErr.ExitCode() != 2 {
			t.Fatalf("expected usage exit 2 for %v, got %v", args, err)
		}
		if stdout.Len() != 0 {
			t.Fatalf("usage must not write stdout for %v: %q", args, stdout.String())
		}
		for _, line := range strings.Split(stderr.String(), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "testule ") {
				continue
			}
			marker := `.\" CLI: ` + line
			if !strings.Contains(manText, marker) {
				t.Fatalf("man page missing executable usage marker %q", marker)
			}
		}
	}
}

func TestUnixExitCodeAndStreamSeparationConformance(t *testing.T) {
	binary := buildCLI(t)
	validPlan, err := os.ReadFile("../../testdata/valid/minimal.yaml")
	if err != nil {
		t.Fatal(err)
	}
	invalidPlan, err := os.ReadFile("../../testdata/invalid/invalid-disposition.yaml")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		args       []string
		stdin      []byte
		exit       int
		stdoutNeed string
		stderrNeed string
	}{
		{name: "success", args: []string{"validate", "--format=json", "-"}, stdin: validPlan, exit: 0, stdoutNeed: `"valid":true`},
		{name: "usage", args: []string{"gaps", "--subject-revision=rev", "-", "-"}, exit: 2, stderrNeed: "usage:"},
		{name: "invalid-document", args: []string{"validate", "--format=json", "-"}, stdin: invalidPlan, exit: 3, stdoutNeed: `"valid":false`},
		{name: "io", args: []string{"validate", "--format=json", filepath.Join(t.TempDir(), "missing.yaml")}, exit: 4, stdoutNeed: `"valid":false`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			command := exec.Command(binary, tc.args...)
			command.Stdin = bytes.NewReader(tc.stdin)
			var stdout, stderr bytes.Buffer
			command.Stdout = &stdout
			command.Stderr = &stderr
			err := command.Run()
			gotExit := 0
			if err != nil {
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) {
					t.Fatalf("run: %v", err)
				}
				gotExit = exitErr.ExitCode()
			}
			if gotExit != tc.exit {
				t.Fatalf("exit=%d want=%d stdout=%q stderr=%q", gotExit, tc.exit, stdout.String(), stderr.String())
			}
			if tc.stdoutNeed != "" && !strings.Contains(stdout.String(), tc.stdoutNeed) {
				t.Fatalf("stdout missing %q: %q", tc.stdoutNeed, stdout.String())
			}
			if tc.stderrNeed != "" && !strings.Contains(stderr.String(), tc.stderrNeed) {
				t.Fatalf("stderr missing %q: %q", tc.stderrNeed, stderr.String())
			}
			if tc.exit == 3 || tc.exit == 4 {
				if stderr.Len() != 0 {
					t.Fatalf("JSON result diagnostics must stay on stdout, stderr=%q", stderr.String())
				}
			}
		})
	}
}

func TestNormalUsagePathDoesNotPromptOrWaitForTTY(t *testing.T) {
	binary := buildCLI(t)
	command := exec.Command(binary, "validate")
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	done := make(chan error, 1)
	go func() { done <- command.Run() }()
	select {
	case err := <-done:
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) || exitErr.ExitCode() != 2 {
			t.Fatalf("expected usage exit 2, got %v", err)
		}
	case <-time.After(2 * time.Second):
		_ = command.Process.Kill()
		t.Fatal("normal usage path waited for interactive input")
	}

	text := strings.ToLower(stderr.String())
	for _, prompt := range []string{"press enter", "continue?", "password:", "open browser"} {
		if strings.Contains(text, prompt) {
			t.Fatalf("unexpected interactive prompt %q in %q", prompt, stderr.String())
		}
	}
}
