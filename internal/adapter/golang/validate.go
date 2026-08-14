package golang

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	runIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	testPattern  = regexp.MustCompile(`^Test[A-Za-z0-9_]+$`)
	fuzzPattern  = regexp.MustCompile(`^Fuzz[A-Za-z0-9_]+$`)
)

var allowedLevels = map[string]struct{}{
	"unit": {}, "component": {}, "contract": {}, "integration": {}, "system": {}, "endToEnd": {},
}
var allowedBehaviors = map[string]struct{}{
	"positive": {}, "negative": {}, "boundary": {}, "adversarial": {},
}
var allowedGeneration = map[string]struct{}{
	"example": {}, "generated": {}, "property": {}, "fuzz": {}, "model": {}, "aiAssisted": {},
}
var allowedVisibility = map[string]struct{}{
	"blackBox": {}, "grayBox": {}, "whiteBox": {},
}
var allowedQuality = map[string]struct{}{
	"functional": {}, "security": {}, "performance": {}, "resilience": {}, "compatibility": {}, "operational": {},
}

func validateRunConfig(cfg RunConfig) (string, string, error) {
	if cfg.Plan == nil || cfg.Plan.Metadata == nil || cfg.Plan.Subject == nil {
		return "", "", errors.New("validated plan is required")
	}
	if cfg.Operation != OperationTest && cfg.Operation != OperationFuzz {
		return "", "", errors.New("operation must be test or fuzz")
	}
	if strings.TrimSpace(cfg.SubjectRevision) == "" || len(cfg.SubjectRevision) > 256 {
		return "", "", errors.New("subject revision is required and must be at most 256 characters")
	}
	if !runIDPattern.MatchString(cfg.RunID) {
		return "", "", errors.New("run id must match [A-Za-z0-9][A-Za-z0-9._-]{0,63}")
	}
	if strings.TrimSpace(cfg.EnvironmentID) == "" || len(cfg.EnvironmentID) > 256 {
		return "", "", errors.New("environment id is required and must be at most 256 characters")
	}
	if cfg.Timeout <= 0 || cfg.Timeout > 2*time.Minute {
		return "", "", errors.New("timeout must be greater than zero and at most 2m")
	}
	if cfg.Operation == OperationFuzz {
		if cfg.FuzzTime <= 0 || cfg.FuzzTime > 30*time.Second {
			return "", "", errors.New("fuzz time must be greater than zero and at most 30s")
		}
		if cfg.FuzzTime+2*time.Second >= cfg.Timeout {
			return "", "", errors.New("timeout must exceed fuzz time by more than 2s")
		}
	}
	if err := validateCoverage(cfg.Operation, cfg.Coverage); err != nil {
		return "", "", err
	}
	workspace, err := resolveWorkspace(cfg.Workspace)
	if err != nil {
		return "", "", err
	}
	packageDir, err := resolvePackageDir(workspace, cfg.Package)
	if err != nil {
		return "", "", err
	}
	if cfg.Operation == OperationTest && !testPattern.MatchString(cfg.Target) {
		return "", "", errors.New("test target must be an exact Test* identifier")
	}
	if cfg.Operation == OperationFuzz && !fuzzPattern.MatchString(cfg.Target) {
		return "", "", errors.New("fuzz target must be an exact Fuzz* identifier")
	}
	return workspace, packageDir, nil
}

func validateCoverage(operation Operation, coverage Coverage) error {
	if _, ok := allowedLevels[coverage.Level]; !ok {
		return errors.New("level is required and must be a supported Testule test level")
	}
	if coverage.Behavior != "" {
		if _, ok := allowedBehaviors[coverage.Behavior]; !ok {
			return errors.New("behavior is not supported")
		}
	}
	if operation == OperationFuzz {
		if coverage.Generation != "fuzz" {
			return errors.New("fuzz execution must use generation=fuzz")
		}
	} else if _, ok := allowedGeneration[coverage.Generation]; !ok {
		return errors.New("generation is not supported")
	}
	if coverage.Visibility != "" {
		if _, ok := allowedVisibility[coverage.Visibility]; !ok {
			return errors.New("visibility is not supported")
		}
	}
	if coverage.QualityAttribute != "" {
		if _, ok := allowedQuality[coverage.QualityAttribute]; !ok {
			return errors.New("quality attribute is not supported")
		}
	}
	return nil
}

func resolveWorkspace(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", errors.New("workspace is required")
	}
	abs, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("resolve workspace: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("resolve workspace symlinks: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", errors.New("workspace must be an existing directory")
	}
	goModInfo, err := os.Lstat(filepath.Join(resolved, "go.mod"))
	if err != nil || !goModInfo.Mode().IsRegular() || goModInfo.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("workspace must contain a regular non-symlink go.mod")
	}
	return resolved, nil
}

func resolvePackageDir(workspace, packageValue string) (string, error) {
	if packageValue == "" {
		packageValue = "."
	}
	if packageValue != "." {
		if !strings.HasPrefix(packageValue, "./") || strings.Contains(packageValue, "...") || strings.ContainsAny(packageValue, "\\\x00\r\n\t") {
			return "", errors.New("package must be . or an exact ./relative/package path")
		}
	}
	rel := strings.TrimPrefix(packageValue, "./")
	if packageValue == "." {
		rel = "."
	}
	clean := filepath.Clean(rel)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || filepath.IsAbs(clean) {
		return "", errors.New("package path must remain within the workspace")
	}
	candidate := filepath.Join(workspace, clean)
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", fmt.Errorf("resolve package path: %w", err)
	}
	if !within(workspace, resolved) {
		return "", errors.New("package path escapes the workspace")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", errors.New("package path must resolve to an existing directory")
	}
	return resolved, nil
}

func within(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
