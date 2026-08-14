package golang

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hackelia-micrantha/testule/internal/evidence"
)

type goTestEvent struct {
	Action string `json:"Action"`
	Test   string `json:"Test"`
	Output string `json:"Output"`
}

func inspectRunOutput(stdout []byte, target string) (bool, []string) {
	observed := false
	seen := map[string]struct{}{}
	var reproducers []string
	prefix := filepath.ToSlash(filepath.Join("testdata", "fuzz", target)) + "/"

	for _, line := range strings.Split(string(stdout), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var event goTestEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		if event.Test == target {
			observed = true
		}
		for _, outputLine := range strings.Split(event.Output, "\n") {
			const marker = "Failing input written to "
			idx := strings.Index(outputLine, marker)
			if idx < 0 {
				continue
			}
			rel := strings.TrimSpace(outputLine[idx+len(marker):])
			rel = filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))
			if !strings.HasPrefix(rel, prefix) {
				continue
			}
			name := strings.TrimPrefix(rel, prefix)
			if name == "" || strings.Contains(name, "/") || name == "." || name == ".." {
				continue
			}
			if _, ok := seen[rel]; ok {
				continue
			}
			seen[rel] = struct{}{}
			reproducers = append(reproducers, rel)
		}
	}
	return observed, reproducers
}

func collectReportedReproducers(packageDir, target, artifactRoot string, reported []string) ([]evidence.Artifact, error) {
	prefix := filepath.ToSlash(filepath.Join("testdata", "fuzz", target)) + "/"
	artifacts := make([]evidence.Artifact, 0, len(reported))
	for _, rel := range reported {
		rel = filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))
		if !strings.HasPrefix(rel, prefix) {
			return nil, errors.New("reported fuzz reproducer is outside the target corpus")
		}
		name := strings.TrimPrefix(rel, prefix)
		if name == "" || strings.Contains(name, "/") || name == "." || name == ".." {
			return nil, errors.New("reported fuzz reproducer has an invalid name")
		}
		source := filepath.Join(packageDir, filepath.FromSlash(rel))
		if !within(packageDir, source) {
			return nil, errors.New("reported fuzz reproducer escapes package directory")
		}
		data, err := readRegularFile(source)
		if err != nil {
			return nil, fmt.Errorf("read reported fuzz reproducer %q: %w", source, err)
		}
		artifact, err := writeResultArtifact(artifactRoot, filepath.ToSlash(filepath.Join("reproducers", name)), "fuzz-reproducer", "application/vnd.go.fuzz-corpus", data)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
		if err := os.Remove(source); err != nil {
			return nil, fmt.Errorf("remove reported workspace fuzz reproducer %q: %w", source, err)
		}
	}
	return artifacts, nil
}
