package golang

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/hackelia-micrantha/testule/internal/evidence"
	"github.com/hackelia-micrantha/testule/internal/plan"
)

func Run(ctx context.Context, cfg RunConfig) (Result, error) {
	workspace, packageDir, err := validateRunConfig(cfg)
	if err != nil {
		return Result{}, err
	}
	artifactRoot, err := createArtifactDir(workspace, cfg.RunID)
	if err != nil {
		return Result{}, err
	}
	env, cleanup, err := isolatedGoEnv(artifactRoot)
	if err != nil {
		return Result{}, err
	}
	defer cleanup()

	fingerprint, err := plan.Fingerprint(cfg.Plan)
	if err != nil {
		return Result{}, err
	}
	intendedArgs := buildRunArgs(cfg)
	binary, lookErr := exec.LookPath("go")
	if lookErr != nil {
		record := buildEvidence(cfg, fingerprint, "unavailable", intendedArgs, commandResult{exitCode: -1}, "unsupported", nil)
		return persistResult(artifactRoot, record, "unsupported", nil, nil)
	}

	version, err := goVersion(ctx, binary, workspace, env)
	if err != nil {
		return Result{}, err
	}

	listed, listResult, err := targetExists(ctx, binary, workspace, env, cfg.Package, cfg.Target, cfg.Timeout)
	if err != nil {
		return Result{}, err
	}
	if !listed {
		status := "unsupported"
		if listResult.exitCode != 0 || listResult.timedOut {
			status = "failed"
		}
		record := buildEvidence(cfg, fingerprint, version, buildListArgs(cfg.Package, cfg.Target), listResult, status, nil)
		return persistResult(artifactRoot, record, status, listResult.stdout, listResult.stderr)
	}

	var before map[string]struct{}
	if cfg.Operation == OperationFuzz {
		before, err = snapshotCorpus(packageDir, cfg.Target)
		if err != nil {
			return Result{}, err
		}
	}

	runCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()
	result, err := runCommand(runCtx, binary, intendedArgs[1:], workspace, env)
	if err != nil {
		return Result{}, err
	}
	status := "passed"
	if result.exitCode != 0 || result.timedOut {
		status = "failed"
	}

	var reproducers []evidence.Artifact
	if cfg.Operation == OperationFuzz && status == "failed" {
		reproducers, err = collectNewReproducers(packageDir, cfg.Target, artifactRoot, before)
		if err != nil {
			return Result{}, err
		}
	}
	record := buildEvidence(cfg, fingerprint, version, intendedArgs, result, status, reproducers)
	return persistResult(artifactRoot, record, status, result.stdout, result.stderr)
}

func buildRunArgs(cfg RunConfig) []string {
	args := []string{"go", "test", "-json", "-count=1", "-parallel=1", "-timeout=" + cfg.Timeout.String()}
	if cfg.Operation == OperationTest {
		args = append(args, "-run=^"+regexp.QuoteMeta(cfg.Target)+"$", cfg.Package)
		return args
	}
	args = append(args,
		"-run=^$",
		"-fuzz=^"+regexp.QuoteMeta(cfg.Target)+"$",
		"-fuzztime="+cfg.FuzzTime.String(),
		cfg.Package,
	)
	return args
}

func buildListArgs(packageValue, target string) []string {
	return []string{"go", "test", "-list=^" + regexp.QuoteMeta(target) + "$", packageValue}
}

func targetExists(ctx context.Context, binary, workspace string, env []string, packageValue, target string, timeout time.Duration) (bool, commandResult, error) {
	listCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	command := buildListArgs(packageValue, target)
	args := command[1:]
	result, err := runCommand(listCtx, binary, args, workspace, env)
	if err != nil {
		return false, result, err
	}
	if result.exitCode != 0 || result.timedOut {
		return false, result, nil
	}
	for _, line := range strings.Split(string(result.stdout), "\n") {
		if strings.TrimSpace(line) == target {
			return true, result, nil
		}
	}
	return false, result, nil
}

func goVersion(ctx context.Context, binary, workspace string, env []string) (string, error) {
	versionCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	result, err := runCommand(versionCtx, binary, []string{"version"}, workspace, env)
	if err != nil {
		return "", err
	}
	if result.exitCode != 0 || result.timedOut {
		return "", errors.New("go version failed")
	}
	fields := strings.Fields(string(result.stdout))
	if len(fields) < 3 {
		return "", errors.New("unexpected go version output")
	}
	return fields[2], nil
}

func buildEvidence(cfg RunConfig, fingerprint, version string, command []string, result commandResult, status string, artifacts []evidence.Artifact) *evidence.Evidence {
	coverage := evidence.Coverage{Levels: []string{cfg.Coverage.Level}, Generation: []string{cfg.Coverage.Generation}}
	if cfg.Coverage.Behavior != "" {
		coverage.Behaviors = []string{cfg.Coverage.Behavior}
	}
	if cfg.Coverage.Visibility != "" {
		coverage.Visibility = []string{cfg.Coverage.Visibility}
	}
	if cfg.Coverage.QualityAttribute != "" {
		coverage.QualityAttributes = []string{cfg.Coverage.QualityAttribute}
	}
	return &evidence.Evidence{
		APIVersion:  cfg.Plan.APIVersion,
		Kind:        evidence.Kind,
		Metadata:    &evidence.Metadata{Name: "go-" + string(cfg.Operation) + "-" + cfg.RunID},
		Plan:        &evidence.PlanReference{Name: cfg.Plan.Metadata.Name, Fingerprint: fingerprint},
		Subject:     &evidence.Subject{Component: cfg.Plan.Subject.Component, Revision: cfg.SubjectRevision},
		Environment: &evidence.Environment{ID: cfg.EnvironmentID},
		Provenance:  &evidence.Provenance{Producer: AdapterID, RunID: cfg.RunID},
		Execution: &evidence.Execution{
			Adapter:         AdapterID,
			Operation:       string(cfg.Operation),
			Tool:            "go",
			ToolVersion:     version,
			Package:         cfg.Package,
			Target:          cfg.Target,
			Command:         append([]string(nil), command...),
			ExitCode:        result.exitCode,
			DurationMillis:  result.duration.Milliseconds(),
			TimedOut:        result.timedOut,
			OutputTruncated: result.truncated,
		},
		Artifacts: artifacts,
		Observations: []evidence.Observation{{
			ID:       "go-" + string(cfg.Operation) + ":" + cfg.Target,
			Status:   status,
			Coverage: coverage,
		}},
	}
}

func persistResult(artifactRoot string, record *evidence.Evidence, status string, stdout, stderr []byte) (Result, error) {
	artifacts := append([]evidence.Artifact(nil), record.Artifacts...)
	if stdout != nil {
		artifact, err := writeResultArtifact(artifactRoot, "results/stdout.log", "stdout", "text/plain", stdout)
		if err != nil {
			return Result{}, err
		}
		artifacts = append(artifacts, artifact)
	}
	if stderr != nil {
		artifact, err := writeResultArtifact(artifactRoot, "results/stderr.log", "stderr", "text/plain", stderr)
		if err != nil {
			return Result{}, err
		}
		artifacts = append(artifacts, artifact)
	}
	record.Artifacts = artifacts
	diagnostics := evidence.Validate(record)
	diagnostics = append(diagnostics, evidence.ValidateExecutionArtifacts(record)...)
	if len(diagnostics) != 0 {
		return Result{}, fmt.Errorf("generated invalid evidence: %v", diagnostics)
	}
	evidencePath := filepath.Join(artifactRoot, "evidence.json")
	f, err := os.OpenFile(evidencePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
	if err != nil {
		return Result{}, err
	}
	encoder := json.NewEncoder(f)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	encodeErr := encoder.Encode(record)
	closeErr := f.Close()
	if encodeErr != nil {
		return Result{}, encodeErr
	}
	if closeErr != nil {
		return Result{}, closeErr
	}
	return Result{Evidence: record, EvidencePath: evidencePath, ArtifactDir: artifactRoot, Status: status}, nil
}
