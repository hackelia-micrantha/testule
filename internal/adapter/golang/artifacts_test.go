package golang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hackelia-micrantha/testule/internal/evidence"
)

func TestWriteResultArtifactRejectsSymlinkedResultDirectory(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "results")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := writeResultArtifact(root, "results/stdout.log", "stdout", "text/plain", []byte("secret")); err == nil {
		t.Fatal("expected symlinked artifact directory to be rejected")
	}
	if _, err := os.Stat(filepath.Join(outside, "stdout.log")); !os.IsNotExist(err) {
		t.Fatalf("artifact escaped through symlink: %v", err)
	}
}

func TestWriteResultArtifactDoesNotOverwritePreexistingFile(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "results"), 0o750); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "results", "stdout.log")
	if err := os.WriteFile(path, []byte("attacker"), 0o640); err != nil {
		t.Fatal(err)
	}
	if _, err := writeResultArtifact(root, "results/stdout.log", "stdout", "text/plain", []byte("trusted")); err == nil {
		t.Fatal("expected pre-existing artifact file to be rejected")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "attacker" {
		t.Fatalf("pre-existing file was overwritten: %q", data)
	}
}

func TestCollectNewReproducerRejectsSymlinkCorpusEntry(t *testing.T) {
	packageDir := t.TempDir()
	corpusDir := filepath.Join(packageDir, "testdata", "fuzz", "FuzzCrash")
	if err := os.MkdirAll(corpusDir, 0o750); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("not fuzz data"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(corpusDir, "newcase")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	artifactRoot := filepath.Join(packageDir, "artifacts")
	if err := os.Mkdir(artifactRoot, 0o750); err != nil {
		t.Fatal(err)
	}
	_, err := collectNewReproducers(packageDir, "FuzzCrash", artifactRoot, corpusSnapshot{entries: map[string]struct{}{}})
	if err == nil || !strings.Contains(err.Error(), "regular non-symlink") {
		t.Fatalf("expected symlink corpus rejection, got %v", err)
	}
}

func TestArtifactAbsolutePathRejectsSymlinkEvidencePath(t *testing.T) {
	root := t.TempDir()
	actual := filepath.Join(root, "actual.json")
	if err := os.WriteFile(actual, []byte("{}"), 0o640); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(root, "evidence.json")
	if err := os.Symlink(actual, linked); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	_, err := artifactAbsolutePath(linked, evidence.Artifact{Path: "reproducers/case", SHA256: "sha256:" + strings.Repeat("a", 64)})
	if err == nil {
		t.Fatal("expected symlink evidence path rejection")
	}
}
