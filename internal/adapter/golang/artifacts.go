package golang

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hackelia-micrantha/testule/internal/evidence"
)

func createArtifactDir(workspace, runID string) (string, error) {
	root := filepath.Join(workspace, ".testule", "artifacts", runID)
	if !within(workspace, root) {
		return "", errors.New("artifact directory escapes workspace")
	}
	if _, err := os.Lstat(root); err == nil {
		return "", errors.New("artifact directory already exists for run id")
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := mkdirNoSymlink(workspace, filepath.Join(".testule", "artifacts", runID)); err != nil {
		return "", err
	}
	return root, nil
}

func mkdirNoSymlink(root, rel string) error {
	current := root
	for _, part := range strings.Split(filepath.Clean(rel), string(filepath.Separator)) {
		if part == "." || part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		switch {
		case err == nil:
			if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return fmt.Errorf("unsafe artifact path component %q", current)
			}
		case os.IsNotExist(err):
			if err := os.Mkdir(current, 0o750); err != nil {
				return err
			}
		default:
			return err
		}
	}
	return nil
}

func writeResultArtifact(root, rel, role, mediaType string, data []byte) (evidence.Artifact, error) {
	path := filepath.Join(root, filepath.FromSlash(rel))
	if !within(root, path) {
		return evidence.Artifact{}, errors.New("artifact path escapes run directory")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return evidence.Artifact{}, err
	}
	if err := os.WriteFile(path, data, 0o640); err != nil {
		return evidence.Artifact{}, err
	}
	digest := sha256.Sum256(data)
	return evidence.Artifact{
		Name:      filepath.Base(path),
		Role:      role,
		Path:      filepath.ToSlash(rel),
		SHA256:    "sha256:" + hex.EncodeToString(digest[:]),
		MediaType: mediaType,
	}, nil
}

func fileDigest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

func snapshotCorpus(packageDir, target string) (map[string]struct{}, error) {
	dir := filepath.Join(packageDir, "testdata", "fuzz", target)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return map[string]struct{}{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			result[entry.Name()] = struct{}{}
		}
	}
	return result, nil
}

func collectNewReproducers(packageDir, target, artifactRoot string, before map[string]struct{}) ([]evidence.Artifact, error) {
	dir := filepath.Join(packageDir, "testdata", "fuzz", target)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if _, existed := before[entry.Name()]; !existed {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	artifacts := make([]evidence.Artifact, 0, len(names))
	for _, name := range names {
		source := filepath.Join(dir, name)
		data, err := os.ReadFile(source)
		if err != nil {
			return nil, err
		}
		artifact, err := writeResultArtifact(artifactRoot, filepath.ToSlash(filepath.Join("reproducers", name)), "fuzz-reproducer", "application/vnd.go.fuzz-corpus", data)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
		if err := os.Remove(source); err != nil {
			return nil, fmt.Errorf("remove workspace fuzz reproducer %q: %w", source, err)
		}
	}
	cleanupEmptyCorpusDirs(packageDir, target)
	return artifacts, nil
}

func cleanupEmptyCorpusDirs(packageDir, target string) {
	_ = os.Remove(filepath.Join(packageDir, "testdata", "fuzz", target))
	_ = os.Remove(filepath.Join(packageDir, "testdata", "fuzz"))
	_ = os.Remove(filepath.Join(packageDir, "testdata"))
}

func artifactAbsolutePath(evidencePath string, artifact evidence.Artifact) (string, error) {
	rootAbs, err := filepath.Abs(filepath.Dir(evidencePath))
	if err != nil {
		return "", err
	}
	root, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", err
	}
	clean := filepath.Clean(filepath.FromSlash(artifact.Path))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("artifact path escapes evidence directory")
	}
	path := filepath.Join(root, clean)
	resolvedParent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	if !within(root, resolvedParent) {
		return "", errors.New("artifact parent escapes evidence directory")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", errors.New("artifact must be a regular non-symlink file")
	}
	digest, err := fileDigest(path)
	if err != nil {
		return "", err
	}
	if digest != artifact.SHA256 {
		return "", errors.New("artifact digest does not match evidence")
	}
	return path, nil
}
