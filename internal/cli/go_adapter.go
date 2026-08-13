package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	goadapter "github.com/hackelia-micrantha/testule/internal/adapter/golang"
	"github.com/hackelia-micrantha/testule/internal/evidence"
	"github.com/hackelia-micrantha/testule/internal/plan"
)

const (
	ExitAdapterFailure = 6
	ExitUnsupported    = 7
)

func runGo(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printGoUsage(stderr)
		return ExitUsage
	}
	switch args[0] {
	case "test":
		return runGoExecute(goadapter.OperationTest, args[1:], stdout, stderr)
	case "fuzz":
		return runGoExecute(goadapter.OperationFuzz, args[1:], stdout, stderr)
	case "replay":
		return runGoReplay(args[1:], stdout, stderr)
	case "promote":
		return runGoPromote(args[1:], stdout, stderr)
	default:
		printGoUsage(stderr)
		return ExitUsage
	}
}

func runGoExecute(operation goadapter.Operation, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("testule go "+string(operation), flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var planPath, revision, workspace, packageValue, target, environmentID, runID string
	var level, behavior, generation, visibility, quality string
	timeout := 30 * time.Second
	fuzzTime := time.Second
	fs.StringVar(&planPath, "plan", "", "TestPlan path")
	fs.StringVar(&revision, "subject-revision", "", "subject revision")
	fs.StringVar(&workspace, "workspace", "", "Go module workspace")
	fs.StringVar(&packageValue, "package", ".", "exact local Go package")
	fs.StringVar(&target, "target", "", "exact Test* or Fuzz* target")
	fs.StringVar(&environmentID, "environment", "", "environment identity")
	fs.StringVar(&runID, "run-id", "", "stable run identity")
	fs.StringVar(&level, "level", "", "Testule test level")
	fs.StringVar(&behavior, "behavior", "", "Testule behavior")
	fs.StringVar(&visibility, "visibility", "", "Testule visibility")
	fs.StringVar(&quality, "quality", "", "Testule quality attribute")
	fs.DurationVar(&timeout, "timeout", timeout, "wall-clock timeout")
	if operation == goadapter.OperationTest {
		generation = "example"
		fs.StringVar(&generation, "generation", generation, "Testule generation strategy")
	} else {
		generation = "fuzz"
		fs.DurationVar(&fuzzTime, "fuzztime", fuzzTime, "bounded native fuzz duration")
	}
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 || planPath == "" || revision == "" || workspace == "" || target == "" || environmentID == "" || runID == "" || level == "" {
		printGoUsage(stderr)
		return ExitUsage
	}

	p, diagnostics, exit := loadPlan(planPath)
	if exit != ExitOK {
		if err := writeResult(stdout, stderr, "text", result{Valid: false, Source: planPath, Diagnostics: diagnostics}); err != nil {
			fmt.Fprintln(stderr, err)
			return ExitInternal
		}
		return exit
	}
	adapterResult, err := goadapter.Run(context.Background(), goadapter.RunConfig{
		Operation: operation, Plan: p, SubjectRevision: revision, Workspace: workspace,
		Package: packageValue, Target: target, EnvironmentID: environmentID, RunID: runID,
		Timeout: timeout, FuzzTime: fuzzTime,
		Coverage: goadapter.Coverage{Level: level, Behavior: behavior, Generation: generation, Visibility: visibility, QualityAttribute: quality},
	})
	if err != nil {
		fmt.Fprintf(stderr, "go adapter: %v\n", err)
		return ExitIO
	}
	if _, err := fmt.Fprintf(stdout, "status: %s\nevidence: %s\n", adapterResult.Status, adapterResult.EvidencePath); err != nil {
		fmt.Fprintln(stderr, err)
		return ExitInternal
	}
	return adapterStatusExit(adapterResult.Status)
}

func runGoReplay(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("testule go replay", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var evidencePath, revision, workspace, environmentID, runID string
	timeout := 30 * time.Second
	fs.StringVar(&evidencePath, "evidence", "", "source fuzz Evidence path")
	fs.StringVar(&revision, "subject-revision", "", "subject revision")
	fs.StringVar(&workspace, "workspace", "", "Go module workspace")
	fs.StringVar(&environmentID, "environment", "", "replay environment identity")
	fs.StringVar(&runID, "run-id", "", "replay run identity")
	fs.DurationVar(&timeout, "timeout", timeout, "wall-clock timeout")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 || evidencePath == "" || revision == "" || workspace == "" || environmentID == "" || runID == "" {
		printGoUsage(stderr)
		return ExitUsage
	}
	record, exit := loadEvidence(evidencePath, stdout, stderr)
	if exit != ExitOK {
		return exit
	}
	adapterResult, err := goadapter.Replay(context.Background(), goadapter.ReplayConfig{
		SourceEvidence: record, EvidencePath: evidencePath, SubjectRevision: revision,
		Workspace: workspace, EnvironmentID: environmentID, RunID: runID, Timeout: timeout,
	})
	if err != nil {
		fmt.Fprintf(stderr, "go replay: %v\n", err)
		return ExitIO
	}
	if _, err := fmt.Fprintf(stdout, "status: %s\nevidence: %s\n", adapterResult.Status, adapterResult.EvidencePath); err != nil {
		fmt.Fprintln(stderr, err)
		return ExitInternal
	}
	return adapterStatusExit(adapterResult.Status)
}

func runGoPromote(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("testule go promote", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var evidencePath, revision, workspace string
	fs.StringVar(&evidencePath, "evidence", "", "source fuzz Evidence path")
	fs.StringVar(&revision, "subject-revision", "", "subject revision")
	fs.StringVar(&workspace, "workspace", "", "Go module workspace")
	if err := fs.Parse(args); err != nil || fs.NArg() != 0 || evidencePath == "" || revision == "" || workspace == "" {
		printGoUsage(stderr)
		return ExitUsage
	}
	record, exit := loadEvidence(evidencePath, stdout, stderr)
	if exit != ExitOK {
		return exit
	}
	destination, err := goadapter.Promote(goadapter.PromoteConfig{
		SourceEvidence: record, EvidencePath: evidencePath, SubjectRevision: revision, Workspace: workspace,
	})
	if err != nil {
		fmt.Fprintf(stderr, "go promote: %v\n", err)
		return ExitIO
	}
	if _, err := fmt.Fprintf(stdout, "promoted: %s\n", destination); err != nil {
		fmt.Fprintln(stderr, err)
		return ExitInternal
	}
	return ExitOK
}

func loadEvidence(path string, stdout, stderr io.Writer) (*evidence.Evidence, int) {
	data, err := readBounded(path, evidence.MaxEvidenceBytes, "evidence")
	if err != nil {
		if writeErr := writeResult(stdout, stderr, "text", result{Valid: false, Source: path, Diagnostics: []plan.Diagnostic{{Code: "io_error", Message: err.Error()}}}); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
			return nil, ExitInternal
		}
		return nil, ExitIO
	}
	record, diagnostics := evidence.Decode(data)
	if len(diagnostics) != 0 {
		if writeErr := writeResult(stdout, stderr, "text", result{Valid: false, Source: path, Diagnostics: diagnostics}); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
			return nil, ExitInternal
		}
		return nil, ExitInvalidPlan
	}
	return record, ExitOK
}

func adapterStatusExit(status string) int {
	switch status {
	case "passed":
		return ExitOK
	case "unsupported":
		return ExitUnsupported
	case "failed":
		return ExitAdapterFailure
	default:
		return ExitInternal
	}
}

func printGoUsage(w io.Writer) {
	fmt.Fprintln(w, "usage:")
	fmt.Fprintln(w, "  testule go test --plan <plan> --subject-revision <rev> --workspace <dir> --package <./pkg> --target <TestName> --level <level> --environment <id> --run-id <id> [--generation example] [--behavior <behavior>] [--visibility <visibility>] [--quality <attribute>] [--timeout 30s]")
	fmt.Fprintln(w, "  testule go fuzz --plan <plan> --subject-revision <rev> --workspace <dir> --package <./pkg> --target <FuzzName> --level <level> --environment <id> --run-id <id> [--behavior <behavior>] [--visibility <visibility>] [--quality <attribute>] [--fuzztime 1s] [--timeout 30s]")
	fmt.Fprintln(w, "  testule go replay --evidence <evidence.json> --subject-revision <rev> --workspace <dir> --environment <id> --run-id <id> [--timeout 30s]")
	fmt.Fprintln(w, "  testule go promote --evidence <evidence.json> --subject-revision <rev> --workspace <dir>")
}
