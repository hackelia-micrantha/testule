package golang

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hackelia-micrantha/testule/internal/evidence"
)

func verifyRecordedArtifacts(root string, artifacts []evidence.Artifact) error {
	if err := ensureDirectoryNoSymlink(root); err != nil {
		return err
	}
	for _, artifact := range artifacts {
		clean := filepath.Clean(filepath.FromSlash(artifact.Path))
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return errors.New("recorded artifact path escapes run directory")
		}
		path := filepath.Join(root, clean)
		if !within(root, path) {
			return errors.New("recorded artifact path escapes run directory")
		}
		if err := ensureDirectoryNoSymlink(filepath.Dir(path)); err != nil {
			return err
		}
		digest, err := fileDigest(path)
		if err != nil {
			return fmt.Errorf("verify artifact %q: %w", artifact.Path, err)
		}
		if digest != artifact.SHA256 {
			return fmt.Errorf("artifact %q digest changed before evidence commit", artifact.Path)
		}
	}
	return nil
}
