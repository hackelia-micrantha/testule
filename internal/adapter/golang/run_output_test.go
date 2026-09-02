package golang

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInspectRunOutputFindsTargetAndReportedReproducer(t *testing.T) {
	stdout := strings.Join([]string{
		`{"Action":"run","Test":"FuzzCrash"}`,
		`{"Action":"output","Test":"FuzzCrash","Output":"    Failing input written to testdata/fuzz/FuzzCrash/abc123\n"}`,
	}, "\n")
	observed, reported := inspectRunOutput([]byte(stdout), "FuzzCrash")
	if !observed {
		t.Fatal("expected target to be observed")
	}
	if len(reported) != 1 || reported[0] != "testdata/fuzz/FuzzCrash/abc123" {
		t.Fatalf("unexpected reported reproducers: %#v", reported)
	}
}

func TestCollectReportedReproducerPreservesUnrelatedConcurrentEntry(t *testing.T) {
	packageDir := t.TempDir()
	corpusDir := filepath.Join(packageDir, "testdata", "fuzz", "FuzzCrash")
	if err := os.MkdirAll(corpusDir, 0o750); err != nil {
		t.Fatal(err)
	}
	reported := filepath.Join(corpusDir, "owned")
	unrelated := filepath.Join(corpusDir, "concurrent")
	if err := os.WriteFile(reported, []byte("go test fuzz v1\n[]byte(\"owned\")\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unrelated, []byte("go test fuzz v1\n[]byte(\"other\")\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	artifactRoot := filepath.Join(t.TempDir(), "artifacts")
	if err := os.MkdirAll(artifactRoot, 0o750); err != nil {
		t.Fatal(err)
	}
	artifacts, err := collectReportedReproducers(packageDir, "FuzzCrash", artifactRoot, []string{"testdata/fuzz/FuzzCrash/owned"})
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 1 || artifacts[0].Name != "owned" {
		t.Fatalf("unexpected artifacts: %#v", artifacts)
	}
	if _, err := os.Stat(reported); !os.IsNotExist(err) {
		t.Fatalf("reported reproducer should be removed after capture: %v", err)
	}
	if data, err := os.ReadFile(unrelated); err != nil || !strings.Contains(string(data), "other") {
		t.Fatalf("unrelated concurrent corpus entry was changed: data=%q err=%v", data, err)
	}
}

func TestRunExecutesPackageLifecycleOnce(t *testing.T) {
	workspace := lifecycleFixtureWorkspace(t)
	result, err := Run(context.Background(), RunConfig{
		Operation:       OperationTest,
		Plan:            adapterPlan(),
		SubjectRevision: "rev-lifecycle",
		Workspace:       workspace,
		Package:         "./sample",
		Target:          "TestPass",
		EnvironmentID:   "test",
		RunID:           "lifecycle-once",
		Timeout:         30 * time.Second,
		Coverage:        Coverage{Level: "unit", Generation: "example"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "passed" {
		t.Fatalf("expected passed status, got %s", result.Status)
	}
	count, err := os.ReadFile(filepath.Join(workspace, "sample", "testmain-count"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(count)) != "1" {
		t.Fatalf("package lifecycle executed %s times; expected once", strings.TrimSpace(string(count)))
	}
}

func TestRunMissingTargetIsUnsupportedWithoutDiscoveryExecution(t *testing.T) {
	workspace := lifecycleFixtureWorkspace(t)
	result, err := Run(context.Background(), RunConfig{
		Operation:       OperationTest,
		Plan:            adapterPlan(),
		SubjectRevision: "rev-missing",
		Workspace:       workspace,
		Package:         "./sample",
		Target:          "TestMissing",
		EnvironmentID:   "test",
		RunID:           "missing-target",
		Timeout:         30 * time.Second,
		Coverage:        Coverage{Level: "unit", Generation: "example"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "unsupported" {
		t.Fatalf("expected unsupported status, got %s", result.Status)
	}
	count, err := os.ReadFile(filepath.Join(workspace, "sample", "testmain-count"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(count)) != "1" {
		t.Fatalf("missing-target lifecycle executed %s times; expected once", strings.TrimSpace(string(count)))
	}
}

func lifecycleFixtureWorkspace(t *testing.T) string {
	t.Helper()
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/lifecycle\n\ngo 1.26\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	packageDir := filepath.Join(workspace, "sample")
	if err := os.Mkdir(packageDir, 0o750); err != nil {
		t.Fatal(err)
	}
	testSource := `package sample

import (
    "os"
    "strconv"
    "strings"
    "testing"
)

func TestMain(m *testing.M) {
    const path = "testmain-count"
    count := 0
    if data, err := os.ReadFile(path); err == nil {
        count, _ = strconv.Atoi(strings.TrimSpace(string(data)))
    }
    _ = os.WriteFile(path, []byte(strconv.Itoa(count+1)), 0o600)
    os.Exit(m.Run())
}

func TestPass(t *testing.T) {}
`
	if err := os.WriteFile(filepath.Join(packageDir, "sample_test.go"), []byte(testSource), 0o640); err != nil {
		t.Fatal(err)
	}
	return workspace
}
