package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"

	"github.com/hackelia-micrantha/testule/internal/evidence"
	"github.com/hackelia-micrantha/testule/internal/gap"
	"github.com/hackelia-micrantha/testule/internal/plan"
)

const (
	ExitOK          = 0
	ExitInternal    = 1
	ExitUsage       = 2
	ExitInvalidPlan = 3
	ExitIO          = 4
	ExitGaps        = 5
)

type result struct {
	Valid       bool              `json:"valid"`
	Source      string            `json:"source"`
	Diagnostics []plan.Diagnostic `json:"diagnostics"`
}

// Run executes a Testule CLI command using the process stdin.
func Run(args []string, stdout, stderr io.Writer) int {
	return RunWithInput(args, os.Stdin, stdout, stderr)
}

// RunWithInput executes a Testule CLI command with an explicit stdin source.
// It exists so callers and tests can provide stream input without mutating
// process-global state.
func RunWithInput(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if stdin == nil {
		stdin = strings.NewReader("")
	}
	if len(args) == 0 {
		printUsage(stderr)
		return ExitUsage
	}

	switch args[0] {
	case "validate":
		return runValidate(args[1:], stdin, stdout, stderr)
	case "fingerprint":
		return runFingerprint(args[1:], stdin, stdout, stderr)
	case "gaps":
		return runGaps(args[1:], stdin, stdout, stderr)
	case "go":
		return runGo(args[1:], stdout, stderr)
	default:
		printUsage(stderr)
		return ExitUsage
	}
}

func runValidate(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	format, path, ok := parseValidateArgs(args)
	if !ok {
		printUsage(stderr)
		return ExitUsage
	}

	p, diagnostics, exit := loadPlan(path, stdin)
	if exit != ExitOK {
		if writeErr := writeResult(stdout, stderr, format, result{Valid: false, Source: path, Diagnostics: diagnostics}); writeErr != nil {
			return handleOutputError(stderr, writeErr)
		}
		return exit
	}
	_ = p

	if err := writeResult(stdout, stderr, format, result{Valid: true, Source: path, Diagnostics: []plan.Diagnostic{}}); err != nil {
		return handleOutputError(stderr, err)
	}
	return ExitOK
}

func runFingerprint(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) != 1 || (args[0] != "-" && strings.HasPrefix(args[0], "-")) {
		printUsage(stderr)
		return ExitUsage
	}
	path := args[0]
	p, diagnostics, exit := loadPlan(path, stdin)
	if exit != ExitOK {
		if writeErr := writeResult(stdout, stderr, "text", result{Valid: false, Source: path, Diagnostics: diagnostics}); writeErr != nil {
			return handleOutputError(stderr, writeErr)
		}
		return exit
	}

	fingerprint, err := plan.Fingerprint(p)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return ExitInternal
	}
	if _, err := fmt.Fprintln(stdout, fingerprint); err != nil {
		return handleOutputError(stderr, err)
	}
	return ExitOK
}

func runGaps(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	format, revision, planPath, evidencePaths, ok := parseGapArgs(args)
	if !ok {
		printUsage(stderr)
		return ExitUsage
	}

	p, diagnostics, exit := loadPlan(planPath, stdin)
	if exit != ExitOK {
		if writeErr := writeResult(stdout, stderr, format, result{Valid: false, Source: planPath, Diagnostics: diagnostics}); writeErr != nil {
			return handleOutputError(stderr, writeErr)
		}
		return exit
	}

	records := make([]*evidence.Evidence, 0, len(evidencePaths))
	for _, path := range evidencePaths {
		data, err := readBounded(path, stdin, evidence.MaxEvidenceBytes, "evidence")
		if err != nil {
			diagnostic := plan.Diagnostic{Code: "io_error", Message: err.Error()}
			if writeErr := writeResult(stdout, stderr, format, result{Valid: false, Source: path, Diagnostics: []plan.Diagnostic{diagnostic}}); writeErr != nil {
				return handleOutputError(stderr, writeErr)
			}
			return ExitIO
		}
		record, evidenceDiagnostics := evidence.Decode(data)
		if len(evidenceDiagnostics) != 0 {
			if writeErr := writeResult(stdout, stderr, format, result{Valid: false, Source: path, Diagnostics: evidenceDiagnostics}); writeErr != nil {
				return handleOutputError(stderr, writeErr)
			}
			return ExitInvalidPlan
		}
		records = append(records, record)
	}

	report, evaluationDiagnostics := gap.Evaluate(p, records, revision)
	if len(evaluationDiagnostics) != 0 {
		if writeErr := writeResult(stdout, stderr, format, result{Valid: false, Source: planPath, Diagnostics: evaluationDiagnostics}); writeErr != nil {
			return handleOutputError(stderr, writeErr)
		}
		return ExitInvalidPlan
	}

	if err := writeGapReport(stdout, format, report); err != nil {
		return handleOutputError(stderr, err)
	}
	if !report.Complete {
		return ExitGaps
	}
	return ExitOK
}

func parseValidateArgs(args []string) (format, path string, ok bool) {
	format = "text"
	for len(args) > 0 {
		switch {
		case args[0] == "--format" && len(args) >= 2:
			format = args[1]
			args = args[2:]
		case strings.HasPrefix(args[0], "--format="):
			format = strings.TrimPrefix(args[0], "--format=")
			args = args[1:]
		case args[0] != "-" && strings.HasPrefix(args[0], "-"):
			return "", "", false
		default:
			if path != "" {
				return "", "", false
			}
			path = args[0]
			args = args[1:]
		}
	}
	if path == "" || !validFormat(format) {
		return "", "", false
	}
	return format, path, true
}

func parseGapArgs(args []string) (format, revision, planPath string, evidencePaths []string, ok bool) {
	format = "text"
	var paths []string
	for len(args) > 0 {
		switch {
		case args[0] == "--format" && len(args) >= 2:
			format = args[1]
			args = args[2:]
		case strings.HasPrefix(args[0], "--format="):
			format = strings.TrimPrefix(args[0], "--format=")
			args = args[1:]
		case args[0] == "--subject-revision" && len(args) >= 2:
			revision = args[1]
			args = args[2:]
		case strings.HasPrefix(args[0], "--subject-revision="):
			revision = strings.TrimPrefix(args[0], "--subject-revision=")
			args = args[1:]
		case args[0] != "-" && strings.HasPrefix(args[0], "-"):
			return "", "", "", nil, false
		default:
			paths = append(paths, args[0])
			args = args[1:]
		}
	}

	if !validFormat(format) || strings.TrimSpace(revision) == "" || len(paths) == 0 {
		return "", "", "", nil, false
	}
	if len(paths) > 129 || countPath(paths, "-") > 1 {
		return "", "", "", nil, false
	}
	return format, revision, paths[0], paths[1:], true
}

func countPath(paths []string, want string) int {
	count := 0
	for _, path := range paths {
		if path == want {
			count++
		}
	}
	return count
}

func validFormat(format string) bool {
	return format == "text" || format == "json"
}

func loadPlan(path string, stdin io.Reader) (*plan.TestPlan, []plan.Diagnostic, int) {
	data, err := readBounded(path, stdin, plan.MaxPlanBytes, "plan")
	if err != nil {
		return nil, []plan.Diagnostic{{Code: "io_error", Message: err.Error()}}, ExitIO
	}
	p, diagnostics := plan.Decode(data)
	if len(diagnostics) != 0 {
		return nil, diagnostics, ExitInvalidPlan
	}
	return p, nil, ExitOK
}

func readBounded(path string, stdin io.Reader, limit int64, noun string) ([]byte, error) {
	if path == "-" {
		return readBoundedReader(stdin, limit, noun, "stdin")
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}
	defer f.Close()

	return readBoundedReader(f, limit, noun, fmt.Sprintf("%q", path))
}

func readBoundedReader(r io.Reader, limit int64, noun, source string) ([]byte, error) {
	limited := io.LimitReader(r, limit+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", source, err)
	}
	if int64(len(data)) > limit {
		return nil, errors.New(noun + " exceeds maximum supported size")
	}
	return data, nil
}

func handleOutputError(stderr io.Writer, err error) int {
	if isBrokenPipe(err) {
		return ExitOK
	}
	fmt.Fprintln(stderr, err)
	return ExitInternal
}

func isBrokenPipe(err error) bool {
	return errors.Is(err, syscall.EPIPE) || errors.Is(err, io.ErrClosedPipe)
}

func writeResult(stdout, stderr io.Writer, format string, r result) error {
	if r.Diagnostics == nil {
		r.Diagnostics = []plan.Diagnostic{}
	}

	if format == "json" {
		encoder := json.NewEncoder(stdout)
		encoder.SetEscapeHTML(false)
		return encoder.Encode(r)
	}

	if r.Valid {
		_, err := fmt.Fprintf(stdout, "valid: %s\n", r.Source)
		return err
	}
	for _, diagnostic := range r.Diagnostics {
		path := diagnostic.Path
		if path != "" {
			path += ": "
		}
		if _, err := fmt.Fprintf(stderr, "%s %s%s\n", diagnostic.Code, path, diagnostic.Message); err != nil {
			return err
		}
	}
	return nil
}

func writeGapReport(stdout io.Writer, format string, report gap.Report) error {
	if format == "json" {
		encoder := json.NewEncoder(stdout)
		encoder.SetEscapeHTML(false)
		return encoder.Encode(report)
	}

	if _, err := fmt.Fprintf(stdout, "complete: %t\nplan: %s\nfingerprint: %s\nsubject: %s@%s\n",
		report.Complete, report.Plan, report.PlanFingerprint, report.Subject, report.SubjectRevision); err != nil {
		return err
	}
	for _, entry := range report.Entries {
		if _, err := fmt.Fprintf(stdout, "%s.%s [%s] %s", entry.Dimension, entry.Value, entry.Disposition, entry.State); err != nil {
			return err
		}
		if len(entry.Evidence) > 0 {
			if _, err := fmt.Fprintf(stdout, " evidence=%s", strings.Join(entry.Evidence, ",")); err != nil {
				return err
			}
		}
		if entry.Rationale != "" {
			if _, err := fmt.Fprintf(stdout, " rationale=%q", entry.Rationale); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(stdout); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(stdout,
		"summary: satisfied=%d missing=%d unsupported=%d skipped=%d failed=%d inapplicable=%d\n",
		report.Summary.Satisfied,
		report.Summary.Missing,
		report.Summary.Unsupported,
		report.Summary.Skipped,
		report.Summary.Failed,
		report.Summary.Inapplicable,
	)
	return err
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage:")
	fmt.Fprintln(w, "  testule validate [--format text|json] <plan.yaml|->")
	fmt.Fprintln(w, "  testule fingerprint <plan.yaml|->")
	fmt.Fprintln(w, "  testule gaps [--format text|json] --subject-revision <revision> <plan.yaml|-> [evidence.yaml|- ...]")
	fmt.Fprintln(w, "  testule go <test|fuzz|replay|promote> ...")
}
