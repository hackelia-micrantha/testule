package python

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	adaptercontract "github.com/hackelia-micrantha/testule/internal/adapter"
	"github.com/hackelia-micrantha/testule/internal/evidence"
)

const (
	AdapterID      = "python-unittest/v1alpha1"
	MaxOutputBytes = 1 << 20
)

type Adapter struct {
	Executable string
}

func (a Adapter) Describe() adaptercontract.Descriptor {
	return adaptercontract.Descriptor{
		ProtocolVersion: adaptercontract.ProtocolVersion,
		ID:              AdapterID,
		Version:         "v1alpha1",
		Capabilities:    []adaptercontract.Capability{adaptercontract.CapabilityTestExecute},
	}
}

func (a Adapter) Probe(context.Context) adaptercontract.ProbeResult {
	_, err := a.resolveExecutable()
	availability := adaptercontract.Availability{Capability: adaptercontract.CapabilityTestExecute, Available: err == nil}
	if err != nil {
		availability.Reason = "python executable unavailable"
	}
	return adaptercontract.ProbeResult{Adapter: AdapterID, Availability: []adaptercontract.Availability{availability}}
}

func (a Adapter) Invoke(ctx context.Context, invocation adaptercontract.Invocation) adaptercontract.Result {
	if invocation.Capability != adaptercontract.CapabilityTestExecute {
		return adaptercontract.Result{Status: adaptercontract.StatusUnsupported, Diagnostics: []string{"unsupported capability"}}
	}
	workspace, err := validateInvocation(invocation)
	if err != nil {
		return adaptercontract.Result{Status: adaptercontract.StatusInfrastructureFailed, Diagnostics: []string{err.Error()}}
	}
	binary, err := a.resolveExecutable()
	if err != nil {
		return adaptercontract.Result{Status: adaptercontract.StatusUnsupported, Diagnostics: []string{"python executable unavailable"}}
	}

	version, err := pythonVersion(ctx, binary, workspace)
	if err != nil {
		return adaptercontract.Result{Status: adaptercontract.StatusInfrastructureFailed, Diagnostics: []string{err.Error()}}
	}

	args := []string{"-I", "-m", "unittest", invocation.TargetID}
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	started := time.Now()
	command := exec.CommandContext(runCtx, binary, args...)
	command.Dir = workspace
	command.Env = isolatedEnvironment()
	var stdout, stderr cappedBuffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err = command.Run()
	duration := time.Since(started)
	timedOut := runCtx.Err() == context.DeadlineExceeded
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !strings.Contains(err.Error(), "exit status") || !asExitError(err, &exitErr) {
			if timedOut {
				return adaptercontract.Result{Status: adaptercontract.StatusTimedOut, Diagnostics: []string{"python unittest timed out"}}
			}
			return adaptercontract.Result{Status: adaptercontract.StatusInfrastructureFailed, Diagnostics: []string{fmt.Sprintf("execute unittest: %v", err)}}
		}
		exitCode = exitErr.ExitCode()
	}
	if timedOut {
		return adaptercontract.Result{Status: adaptercontract.StatusTimedOut, Diagnostics: []string{"python unittest timed out"}}
	}

	observationStatus := "passed"
	if exitCode != 0 {
		observationStatus = "failed"
	}
	record := &evidence.Evidence{
		APIVersion:  invocation.Plan.APIVersion,
		Kind:        evidence.Kind,
		Metadata:    &evidence.Metadata{Name: "python-test-" + invocation.RunID},
		Plan:        &evidence.PlanReference{Name: invocation.Plan.Name, Fingerprint: invocation.Plan.Fingerprint},
		Subject:     &evidence.Subject{Component: invocation.Plan.Component, Revision: invocation.SubjectRevision},
		Environment: &evidence.Environment{ID: invocation.EnvironmentID},
		Provenance:  &evidence.Provenance{Producer: AdapterID, RunID: invocation.RunID},
		Execution: &evidence.Execution{
			Adapter: AdapterID, Operation: "test", Tool: "python", ToolVersion: version,
			Target: invocation.TargetID, Command: append([]string{filepath.Base(binary)}, args...),
			ExitCode: exitCode, DurationMillis: duration.Milliseconds(), OutputTruncated: stdout.truncated || stderr.truncated,
		},
		Observations: []evidence.Observation{{ID: "python-test:" + invocation.TargetID, Status: observationStatus, Coverage: coverage(invocation.Coverage)}},
	}
	if diagnostics := append(evidence.Validate(record), evidence.ValidateExecutionArtifacts(record)...); len(diagnostics) != 0 {
		return adaptercontract.Result{Status: adaptercontract.StatusInfrastructureFailed, Diagnostics: []string{fmt.Sprintf("generated invalid evidence: %v", diagnostics)}}
	}
	return adaptercontract.Result{Status: adaptercontract.StatusCompleted, Evidence: record}
}

func (a Adapter) resolveExecutable() (string, error) {
	name := a.Executable
	if name == "" {
		name = "python3"
	}
	return exec.LookPath(name)
}

func validateInvocation(invocation adaptercontract.Invocation) (string, error) {
	if strings.TrimSpace(invocation.TargetID) == "" || utf8.RuneCountInString(invocation.TargetID) > 120 {
		return "", fmt.Errorf("python target ID must contain 1 to 120 Unicode code points")
	}
	for _, r := range invocation.TargetID {
		if unicode.IsControl(r) {
			return "", fmt.Errorf("python target ID must not contain control characters")
		}
	}
	for key := range invocation.AdapterOptions {
		if key != "python.workspace" {
			return "", fmt.Errorf("unsupported Python adapter option %q", key)
		}
	}
	workspace := invocation.AdapterOptions["python.workspace"]
	if workspace == "" {
		return "", fmt.Errorf("python.workspace is required")
	}
	absolute, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("resolve python.workspace: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve python.workspace: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("python.workspace must resolve to a directory")
	}
	return resolved, nil
}

func pythonVersion(ctx context.Context, binary, workspace string) (string, error) {
	versionCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	command := exec.CommandContext(versionCtx, binary, "--version")
	command.Dir = workspace
	command.Env = isolatedEnvironment()
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("python --version: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func isolatedEnvironment() []string {
	env := []string{"PYTHONNOUSERSITE=1", "PYTHONDONTWRITEBYTECODE=1"}
	if path := os.Getenv("PATH"); path != "" {
		env = append(env, "PATH="+path)
	}
	return env
}

func coverage(value adaptercontract.Coverage) evidence.Coverage {
	result := evidence.Coverage{}
	if value.Level != "" { result.Levels = []string{value.Level} }
	if value.Behavior != "" { result.Behaviors = []string{value.Behavior} }
	if value.Generation != "" { result.Generation = []string{value.Generation} }
	if value.Visibility != "" { result.Visibility = []string{value.Visibility} }
	if value.QualityAttribute != "" { result.QualityAttributes = []string{value.QualityAttribute} }
	return result
}

type cappedBuffer struct {
	buffer    bytes.Buffer
	truncated bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if b.buffer.Len() < MaxOutputBytes {
		remaining := MaxOutputBytes - b.buffer.Len()
		if len(p) > remaining {
			_, _ = b.buffer.Write(p[:remaining])
			b.truncated = true
		} else {
			_, _ = b.buffer.Write(p)
		}
	} else {
		b.truncated = true
	}
	return len(p), nil
}

func asExitError(err error, target **exec.ExitError) bool {
	value, ok := err.(*exec.ExitError)
	if ok {
		*target = value
	}
	return ok
}
