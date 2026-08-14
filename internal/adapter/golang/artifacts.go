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

const MaxArtifactBytes int64 = 1 << 20

type corpusSnapshot struct {
	entries         map[string]struct{}
	testdataExisted bool
	fuzzExisted     bool
	targetExisted   bool
}

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
	return root, ensureDirectoryNoSymlink(root)
}

func mkdirNoSymlink(root, rel string) error {
	if err := ensureDirectoryNoSymlink(root); err != nil {
		return err
	}
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
				return fmt.Errorf("unsafe path component %q", current)
			}
		case os.IsNotExist(err):
			if err := os.Mkdir(current, 0o750); err != nil {
				return err
			}
		default:
			return err
		}
	}
	return ensureDirectoryNoSymlink(current)
}

func ensureDirectoryNoSymlink(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("unsafe directory %q", path)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return err
	}
	if filepath.Clean(resolved) != filepath.Clean(abs) {
		return fmt.Errorf("directory path contains a symlink: %q", path)
	}
	return nil
}

func writeResultArtifact(root, rel, role, mediaType string, data []byte) (evidence.Artifact, error) {
	if int64(len(data)) > MaxArtifactBytes {
		return evidence.Artifact{}, errors.New("artifact exceeds maximum supported size")
	}
	path := filepath.Join(root, filepath.FromSlash(rel))
	if !within(root, path) {
		return evidence.Artifact{}, errors.New("artifact path escapes run directory")
	}
	dirRel, err := filepath.Rel(root, filepath.Dir(path))
	if err != nil {
		return evidence.Artifact{}, err
	}
	if err := mkdirNoSymlink(root, dirRel); err != nil {
		return evidence.Artifact{}, err
	}
	if err := writeExclusiveRegular(path, data); err != nil {
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

func writeExclusiveRegular(path string, data []byte) error {
	if int64(len(data)) > MaxArtifactBytes {
		return errors.New("file exceeds maximum supported size")
	}
	if err := ensureDirectoryNoSymlink(filepath.Dir(path)); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if writeErr != nil {
		_ = os.Remove(path)
		return writeErr
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return closeErr
	}
	return nil
}

func readRegularFile(path string) ([]byte, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, errors.New("file must be a regular non-symlink file")
	}
	if before.Size() > MaxArtifactBytes {
		return nil, errors.New("file exceeds maximum supported size")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return nil, err
	}
	after, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if after.Mode()&os.ModeSymlink != 0 || !after.Mode().IsRegular() || !os.SameFile(opened, after) {
		return nil, errors.New("file changed while being opened")
	}
	return io.ReadAll(io.LimitReader(f, MaxArtifactBytes+1))
}

func fileDigest(path string) (string, error) {
	data, err := readRegularFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func snapshotCorpus(packageDir, target string) (corpusSnapshot, error) {
	snapshot := corpusSnapshot{entries: map[string]struct{}{}}
	var err error
	snapshot.testdataExisted, err = safeDirectoryExists(packageDir, "testdata")
	if err != nil {
		return corpusSnapshot{}, err
	}
	snapshot.fuzzExisted, err = safeDirectoryExists(packageDir, filepath.Join("testdata", "fuzz"))
	if err != nil {
		return corpusSnapshot{}, err
	}
	targetRel := filepath.Join("testdata", "fuzz", target)
	snapshot.targetExisted, err = safeDirectoryExists(packageDir, targetRel)
	if err != nil {
		return corpusSnapshot{}, err
	}
	if !snapshot.targetExisted {
		return snapshot, nil
	}
	dir := filepath.Join(packageDir, targetRel)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return corpusSnapshot{}, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			snapshot.entries[entry.Name()] = struct{}{}
		}
	}
	return snapshot, nil
}

func safeDirectoryExists(root, rel string) (bool, error) {
	if err := ensureDirectoryNoSymlink(root); err != nil {
		return false, err
	}
	current := root
	for _, part := range strings.Split(filepath.Clean(rel), string(filepath.Separator)) {
		if part == "." || part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return false, fmt.Errorf("unsafe directory component %q", current)
		}
	}
	return true, nil
}

func collectNewReproducers(packageDir, target, artifactRoot string, before corpusSnapshot) ([]evidence.Artifact, error) {
	targetRel := filepath.Join("testdata", "fuzz", target)
	exists, err := safeDirectoryExists(packageDir, targetRel)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	dir := filepath.Join(packageDir, targetRel)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if _, existed := before.entries[entry.Name()]; !existed {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	artifacts := make([]evidence.Artifact, 0, len(names))
	for _, name := range names {
		source := filepath.Join(dir, name)
		data, err := readRegularFile(source)
		if err != nil {
			return nil, fmt.Errorf("read fuzz reproducer %q: %w", source, err)
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
	cleanupCreatedCorpusDirs(packageDir, target, before)
	return artifacts, nil
}

func cleanupCreatedCorpusDirs(packageDir, target string, before corpusSnapshot) {
	removeEmptyDirectory := func(path string, existed bool) {
		if existed {
			return
		}
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return
		}
		_ = os.Remove(path)
	}
	removeEmptyDirectory(filepath.Join(packageDir, "testdata", "fuzz", target), before.targetExisted)
	removeEmptyDirectory(filepath.Join(packageDir, "testdata", "fuzz"), before.fuzzExisted)
	removeEmptyDirectory(filepath.Join(packageDir, "testdata"), before.testdataExisted)
}

func artifactAbsolutePath(evidencePath string, artifact evidence.Artifact) (string, error) {
	evidenceInfo, err := os.Lstat(evidencePath)
	if err != nil {
		return "", err
	}
	if evidenceInfo.Mode()&os.ModeSymlink != 0 || !evidenceInfo.Mode().IsRegular() {
		return "", errors.New("evidence path must be a regular non-symlink file")
	}
	rootAbs, err := filepath.Abs(filepath.Dir(evidencePath))
	if err != nil {
		return "", err
	}
	if err := ensureDirectoryNoSymlink(rootAbs); err != nil {
		return "", err
	}
	clean := filepath.Clean(filepath.FromSlash(artifact.Path))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("artifact path escapes evidence directory")
	}
	path := filepath.Join(rootAbs, clean)
	if !within(rootAbs, path) {
		return "", errors.New("artifact path escapes evidence directory")
	}
	if err := ensureDirectoryNoSymlink(filepath.Dir(path)); err != nil {
		return "", err
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
