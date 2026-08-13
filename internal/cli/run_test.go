package cli

import (
	"bytes"
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
