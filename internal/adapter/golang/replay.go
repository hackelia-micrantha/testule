package golang

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/hackelia-micrantha/testule/internal/evidence"
)

func Replay(ctx context.Context, cfg ReplayConfig) (Result, error) {
	source, artifact, err := validateReplaySource(cfg.SourceEvidence, cfg.EvidencePath, cfg.SubjectRevision)
	if err != nil {
		return Result{}, err
	}
	workspace, err := resolveWorkspace(cfg.Workspace)
	if err != nil {
		return Result{}, err
	}
	packageDir, err := resolvePackageDir(workspace, source.Execution.Package)
	if err != nil {
		return Result{}, err
	}
	if !runIDPattern.MatchString(cfg.RunID) {
		return Result{}, errors.New("run id is invalid")
	}
	if cfg.Timeout <= 0 || cfg.Timeout > 2*time.Minute {
		return Result{}, errors.New("timeout must be greater than zero and at most 2m")
	}
	if strings.TrimSpace(cfg.EnvironmentID) == "" || len(cfg.EnvironmentID) > 256 {
		return Result{}, errors.New("environment id is required and must be at most 256 characters")
	}
	artifactRoot, err := createArtifactDir(workspace, cfg.RunID)
	if err != nil {
		return Result{}, err
	}
	env, cleanup, err := isolatedGoEnv()
	if err != nil {
		return Result{}, err
	}
	defer cleanup()

	sourcePath, err := artifactAbsolutePath(cfg.EvidencePath, artifact)
	if err != nil {
		return Result{}, err
	}
	sourceData, err := readRegularFile(sourcePath)
	if err != nil {
		return Result{}, err
	}
	replayArtifact, err := writeResultArtifact(artifactRoot, filepath.ToSlash(filepath.Join("reproducers", artifact.Name)), "fuzz-reproducer", artifact.MediaType, sourceData)
	if err != nil {
		return Result{}, err
	}

	before, err := snapshotCorpus(packageDir, source.Execution.Target)
	if err != nil {
		return Result{}, err
	}
	destinationDir := filepath.Join(packageDir, "testdata", "fuzz", source.Execution.Target)
	if err := mkdirNoSymlink(packageDir, filepath.Join("testdata", "fuzz", source.Execution.Target)); err != nil {
		return Result{}, err
	}
	destination := filepath.Join(destinationDir, artifact.Name)
	created := false
	if _, err := os.Lstat(destination); os.IsNotExist(err) {
		if err := writeExclusiveRegular(destination, sourceData); err != nil {
			return Result{}, err
		}
		created = true
	} else if err != nil {
		return Result{}, err
	} else {
		digest, err := fileDigest(destination)
		if err != nil || digest != artifact.SHA256 {
			return Result{}, errors.New("existing fuzz corpus entry conflicts with reproducer")
		}
	}
	if created {
		defer func() {
			_ = os.Remove(destination)
			cleanupCreatedCorpusDirs(packageDir, source.Execution.Target, before)
		}()
	}

	binary, err := exec.LookPath("go")
	if err != nil {
		return Result{}, errors.New("go tool is unavailable for replay")
	}
	version, err := goVersion(ctx, binary, workspace, env)
	if err != nil {
		return Result{}, err
	}
	args := []string{"go", "test", "-json", "-count=1", "-parallel=1", "-timeout=" + cfg.Timeout.String(), "-run=^" + regexp.QuoteMeta(source.Execution.Target+"/"+artifact.Name) + "$", source.Execution.Package}
	runCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()
	result, err := runCommand(runCtx, binary, args[1:], workspace, env)
	if err != nil {
		return Result{}, err
	}
	status := "passed"
	if result.exitCode != 0 || result.timedOut {
		status = "failed"
	}
	coverage := source.Observations[0].Coverage
	record := &evidence.Evidence{
		APIVersion:  source.APIVersion,
		Kind:        evidence.Kind,
		Metadata:    &evidence.Metadata{Name: "go-replay-" + cfg.RunID},
		Plan:        &evidence.PlanReference{Name: source.Plan.Name, Fingerprint: source.Plan.Fingerprint},
		Subject:     &evidence.Subject{Component: source.Subject.Component, Revision: source.Subject.Revision},
		Environment: &evidence.Environment{ID: cfg.EnvironmentID},
		Provenance:  &evidence.Provenance{Producer: AdapterID, RunID: cfg.RunID, References: []string{"source-evidence:" + filepath.Base(cfg.EvidencePath)}},
		Execution: &evidence.Execution{
			Adapter: AdapterID, Operation: string(OperationReplay), Tool: "go", ToolVersion: version,
			Package: source.Execution.Package, Target: source.Execution.Target, Command: args,
			ExitCode: result.exitCode, DurationMillis: result.duration.Milliseconds(), TimedOut: result.timedOut, OutputTruncated: result.truncated,
		},
		Artifacts: []evidence.Artifact{replayArtifact},
		Observations: []evidence.Observation{{
			ID: "go-replay:" + source.Execution.Target + "/" + artifact.Name, Status: status, Coverage: coverage,
		}},
	}
	return persistResult(artifactRoot, record, status, result.stdout, result.stderr)
}

func Promote(cfg PromoteConfig) (string, error) {
	source, artifact, err := validateReplaySource(cfg.SourceEvidence, cfg.EvidencePath, cfg.SubjectRevision)
	if err != nil {
		return "", err
	}
	workspace, err := resolveWorkspace(cfg.Workspace)
	if err != nil {
		return "", err
	}
	packageDir, err := resolvePackageDir(workspace, source.Execution.Package)
	if err != nil {
		return "", err
	}
	sourcePath, err := artifactAbsolutePath(cfg.EvidencePath, artifact)
	if err != nil {
		return "", err
	}
	if err := mkdirNoSymlink(packageDir, filepath.Join("testdata", "fuzz", source.Execution.Target)); err != nil {
		return "", err
	}
	destination := filepath.Join(packageDir, "testdata", "fuzz", source.Execution.Target, artifact.Name)
	if info, err := os.Lstat(destination); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return "", errors.New("promotion destination is not a regular file")
		}
		digest, err := fileDigest(destination)
		if err != nil {
			return "", err
		}
		if digest == artifact.SHA256 {
			return destination, nil
		}
		return "", errors.New("promotion destination already exists with different content")
	} else if !os.IsNotExist(err) {
		return "", err
	}
	data, err := readRegularFile(sourcePath)
	if err != nil {
		return "", err
	}
	if err := writeExclusiveRegular(destination, data); err != nil {
		return "", err
	}
	return destination, nil
}

func validateReplaySource(source *evidence.Evidence, evidencePath, revision string) (*evidence.Evidence, evidence.Artifact, error) {
	if source == nil || source.Execution == nil || source.Plan == nil || source.Subject == nil || source.Environment == nil {
		return nil, evidence.Artifact{}, errors.New("source evidence is incomplete")
	}
	if source.Execution.Adapter != AdapterID || source.Execution.Operation != string(OperationFuzz) {
		return nil, evidence.Artifact{}, errors.New("source evidence is not a Go fuzz execution")
	}
	if source.Subject.Revision != revision || strings.TrimSpace(revision) == "" {
		return nil, evidence.Artifact{}, errors.New("subject revision does not match source evidence")
	}
	if !fuzzPattern.MatchString(source.Execution.Target) {
		return nil, evidence.Artifact{}, errors.New("source evidence has invalid fuzz target")
	}
	if strings.TrimSpace(evidencePath) == "" {
		return nil, evidence.Artifact{}, errors.New("source evidence path is required")
	}
	var reproducers []evidence.Artifact
	for _, artifact := range source.Artifacts {
		if artifact.Role == "fuzz-reproducer" {
			reproducers = append(reproducers, artifact)
		}
	}
	if len(reproducers) != 1 {
		return nil, evidence.Artifact{}, fmt.Errorf("expected exactly one fuzz reproducer artifact, got %d", len(reproducers))
	}
	if len(source.Observations) != 1 {
		return nil, evidence.Artifact{}, errors.New("source evidence must contain exactly one adapter observation")
	}
	if source.Observations[0].Status != "failed" {
		return nil, evidence.Artifact{}, errors.New("source fuzz evidence must represent a failed observation")
	}
	return source, reproducers[0], nil
}
