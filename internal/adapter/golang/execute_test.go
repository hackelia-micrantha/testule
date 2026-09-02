package golang

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hackelia-micrantha/testule/internal/evidence"
	"github.com/hackelia-micrantha/testule/internal/plan"
)

func TestRunMissingGoToolProducesUnsupportedEvidence(t *testing.T) {
	workspace := fixtureWorkspace(t)
	t.Setenv("PATH", "")
	result, err := Run(context.Background(), RunConfig{
		Operation: OperationTest, Plan: adapterPlan(), SubjectRevision: "rev-1", Workspace: workspace,
		Package: "./sample", Target: "TestPass", EnvironmentID: "test", RunID: "missing-go",
		Timeout: 15 * time.Second, Coverage: Coverage{Level: "unit", Generation: "example"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "unsupported" || result.Evidence.Execution.ToolVersion != "unavailable" {
		t.Fatalf("unexpected unsupported result: %#v", result.Evidence.Execution)
	}
}

func TestRunFuzzFailureRetainsNativeReproducer(t *testing.T) {
	workspace := fixtureWorkspace(t)
	result, err := Run(context.Background(), RunConfig{
		Operation: OperationFuzz, Plan: adapterPlan(), SubjectRevision: "rev-fuzz", Workspace: workspace,
		Package: "./sample", Target: "FuzzCrash", EnvironmentID: "test", RunID: "native-fuzz-failure",
		Timeout: 30 * time.Second, FuzzTime: 500 * time.Millisecond,
		Coverage: Coverage{Level: "unit", Generation: "fuzz"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "failed" {
		t.Fatalf("expected fuzz failure, got %s", result.Status)
	}
	var reproducer *evidence.Artifact
	for i := range result.Evidence.Artifacts {
		if result.Evidence.Artifacts[i].Role == "fuzz-reproducer" {
			reproducer = &result.Evidence.Artifacts[i]
			break
		}
	}
	if reproducer == nil {
		t.Fatalf("native fuzz failure did not retain a reproducer: execution=%#v artifacts=%#v", result.Evidence.Execution, result.Evidence.Artifacts)
	}
	if _, err := os.Stat(filepath.Join(workspace, "sample", "testdata", "fuzz", "FuzzCrash", reproducer.Name)); !os.IsNotExist(err) {
		t.Fatalf("reproducer must not remain promoted in the package corpus: %v", err)
	}
	artifactPath := filepath.Join(result.ArtifactDir, filepath.FromSlash(reproducer.Path))
	if digest, err := fileDigest(artifactPath); err != nil || digest != reproducer.SHA256 {
		t.Fatalf("retained reproducer digest mismatch: digest=%s err=%v", digest, err)
	}
}

func TestCollectNewReproducerCopiesAndRemovesWorkspaceEntry(t *testing.T) {
	workspace := t.TempDir()
	packageDir := filepath.Join(workspace, "sample")
	corpusDir := filepath.Join(packageDir, "testdata", "fuzz", "FuzzCrash")
	if err := os.MkdirAll(corpusDir, 0o750); err != nil {
		t.Fatal(err)
	}
	artifactRoot := filepath.Join(workspace, "artifacts")
	if err := os.MkdirAll(artifactRoot, 0o750); err != nil {
		t.Fatal(err)
	}
	contents := []byte("go test fuzz v1\n[]byte(\"boom\")\n")
	if err := os.WriteFile(filepath.Join(corpusDir, "abc123"), contents, 0o640); err != nil {
		t.Fatal(err)
	}
	artifacts, err := collectNewReproducers(packageDir, "FuzzCrash", artifactRoot, corpusSnapshot{entries: map[string]struct{}{}})
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 1 || artifacts[0].Role != "fuzz-reproducer" {
		t.Fatalf("unexpected artifacts: %#v", artifacts)
	}
	if _, err := os.Stat(filepath.Join(corpusDir, "abc123")); !os.IsNotExist(err) {
		t.Fatalf("workspace reproducer was not removed: %v", err)
	}
}

func TestCollectNewReproducerPreservesPreexistingCorpusDirectories(t *testing.T) {
	packageDir := filepath.Join(t.TempDir(), "sample")
	corpusDir := filepath.Join(packageDir, "testdata", "fuzz", "FuzzCrash")
	if err := os.MkdirAll(corpusDir, 0o750); err != nil {
		t.Fatal(err)
	}
	before, err := snapshotCorpus(packageDir, "FuzzCrash")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corpusDir, "newcase"), []byte("go test fuzz v1\n[]byte(\"boom\")\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	artifactRoot := filepath.Join(filepath.Dir(packageDir), "artifacts")
	if err := os.Mkdir(artifactRoot, 0o750); err != nil {
		t.Fatal(err)
	}
	if _, err := collectNewReproducers(packageDir, "FuzzCrash", artifactRoot, before); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(packageDir, "testdata"), filepath.Join(packageDir, "testdata", "fuzz"), corpusDir} {
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			t.Fatalf("pre-existing corpus directory was removed: %s: %v", path, err)
		}
	}
}

func TestPromoteCopiesVerifiedReproducer(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/x\n\ngo 1.23\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(workspace, "sample"), 0o750); err != nil {
		t.Fatal(err)
	}
	artifactRoot := filepath.Join(workspace, "source")
	if err := os.MkdirAll(filepath.Join(artifactRoot, "reproducers"), 0o750); err != nil {
		t.Fatal(err)
	}
	contents := []byte("go test fuzz v1\n[]byte(\"boom\")\n")
	artifactPath := filepath.Join(artifactRoot, "reproducers", "abc123")
	if err := os.WriteFile(artifactPath, contents, 0o640); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(contents)
	record := replayEvidence("abc123", "sha256:"+hex.EncodeToString(sum[:]))
	evidencePath := filepath.Join(artifactRoot, "evidence.json")
	if err := os.WriteFile(evidencePath, []byte("{}"), 0o640); err != nil {
		t.Fatal(err)
	}
	destination, err := Promote(PromoteConfig{SourceEvidence: record, EvidencePath: evidencePath, SubjectRevision: "rev-1", Workspace: workspace})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(filepath.ToSlash(destination), "sample/testdata/fuzz/FuzzCrash/abc123") {
		t.Fatalf("unexpected promotion destination: %s", destination)
	}
	if digest, _ := fileDigest(destination); digest != record.Artifacts[0].SHA256 {
		t.Fatal("promoted corpus digest mismatch")
	}
}

func TestResolvePackageRejectsTraversal(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "go.mod"), []byte("module example.com/x\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := resolvePackageDir(workspace, "../escape"); err == nil {
		t.Fatal("expected traversal rejection")
	}
}

func fixtureWorkspace(t *testing.T) string {
	t.Helper()
	source, err := filepath.Abs(filepath.Join("..", "..", "..", "testdata", "go-adapter"))
	if err != nil {
		t.Fatal(err)
	}
	workspace := t.TempDir()
	copyFixtureFile(t, filepath.Join(source, "go.mod"), filepath.Join(workspace, "go.mod"))
	if err := os.MkdirAll(filepath.Join(workspace, "sample"), 0o750); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"sample.go", "sample_test.go"} {
		copyFixtureFile(t, filepath.Join(source, "sample", name), filepath.Join(workspace, "sample", name))
	}
	return workspace
}

func copyFixtureFile(t *testing.T, source, destination string) {
	t.Helper()
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, data, 0o640); err != nil {
		t.Fatal(err)
	}
}

func adapterPlan() *plan.TestPlan {
	return &plan.TestPlan{APIVersion: plan.APIVersion, Kind: "TestPlan", Metadata: &plan.Metadata{Name: "adapter"}, Subject: &plan.Subject{Component: "fixture"}, Requirements: &plan.Requirements{}}
}

func replayEvidence(name, digest string) *evidence.Evidence {
	return &evidence.Evidence{
		APIVersion: plan.APIVersion, Kind: evidence.Kind, Metadata: &evidence.Metadata{Name: "source"},
		Plan:    &evidence.PlanReference{Name: "adapter", Fingerprint: "sha256:" + strings.Repeat("a", 64)},
		Subject: &evidence.Subject{Component: "fixture", Revision: "rev-1"}, Environment: &evidence.Environment{ID: "test"},
		Provenance:   &evidence.Provenance{Producer: AdapterID, RunID: "source"},
		Execution:    &evidence.Execution{Adapter: AdapterID, Operation: string(OperationFuzz), Tool: "go", ToolVersion: "go1.23", Scope: "./sample", Target: "FuzzCrash", Command: []string{"go", "test"}, ExitCode: 1},
		Artifacts:    []evidence.Artifact{{Name: name, Role: "fuzz-reproducer", Path: "reproducers/" + name, SHA256: digest, MediaType: "application/vnd.go.fuzz-corpus"}},
		Observations: []evidence.Observation{{ID: "go-fuzz:FuzzCrash", Status: "failed", Coverage: evidence.Coverage{Levels: []string{"unit"}, Generation: []string{"fuzz"}}}},
	}
}

func TestReplayReproducesFailureWithoutPromoting(t *testing.T) {
	workspace := fixtureWorkspace(t)
	sourceRoot := filepath.Join(workspace, ".testule", "source")
	if err := os.MkdirAll(filepath.Join(sourceRoot, "reproducers"), 0o750); err != nil {
		t.Fatal(err)
	}
	contents := []byte("go test fuzz v1\n[]byte(\"boom\")\n")
	sum := sha256.Sum256(contents)
	artifactName := "boomcase"
	if err := os.WriteFile(filepath.Join(sourceRoot, "reproducers", artifactName), contents, 0o640); err != nil {
		t.Fatal(err)
	}
	evidencePath := filepath.Join(sourceRoot, "evidence.json")
	if err := os.WriteFile(evidencePath, []byte("{}"), 0o640); err != nil {
		t.Fatal(err)
	}
	record := replayEvidence(artifactName, "sha256:"+hex.EncodeToString(sum[:]))
	result, err := Replay(context.Background(), ReplayConfig{
		SourceEvidence: record, EvidencePath: evidencePath, SubjectRevision: "rev-1",
		Workspace: workspace, EnvironmentID: "replay-test", RunID: "replay", Timeout: 15 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "failed" || result.Evidence.Execution.ExitCode == 0 {
		t.Fatalf("expected replay to reproduce failure: %#v", result.Evidence.Execution)
	}
	staged := filepath.Join(workspace, "sample", "testdata", "fuzz", "FuzzCrash", artifactName)
	if _, err := os.Stat(staged); !os.IsNotExist(err) {
		t.Fatalf("replay corpus entry should be removed after replay: %v", err)
	}
}
