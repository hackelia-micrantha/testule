package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hackelia-micrantha/testule/internal/plan"
)

const (
	ExitOK          = 0
	ExitInternal    = 1
	ExitUsage       = 2
	ExitInvalidPlan = 3
	ExitIO          = 4
)

type result struct {
	Valid       bool              `json:"valid"`
	Source      string            `json:"source"`
	Diagnostics []plan.Diagnostic `json:"diagnostics"`
}

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "validate" {
		printUsage(stderr)
		return ExitUsage
	}

	format, path, ok := parseValidateArgs(args[1:])
	if !ok {
		printUsage(stderr)
		return ExitUsage
	}

	data, err := readBounded(path)
	if err != nil {
		diagnostic := plan.Diagnostic{Code: "io_error", Message: err.Error()}
		if writeErr := writeResult(stdout, stderr, format, result{Valid: false, Source: path, Diagnostics: []plan.Diagnostic{diagnostic}}); writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
			return ExitInternal
		}
		return ExitIO
	}

	_, diagnostics := plan.Decode(data)
	valid := len(diagnostics) == 0
	if err := writeResult(stdout, stderr, format, result{Valid: valid, Source: path, Diagnostics: diagnostics}); err != nil {
		fmt.Fprintln(stderr, err)
		return ExitInternal
	}
	if !valid {
		return ExitInvalidPlan
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
		case strings.HasPrefix(args[0], "-"):
			return "", "", false
		default:
			if path != "" {
				return "", "", false
			}
			path = args[0]
			args = args[1:]
		}
	}
	if path == "" || (format != "text" && format != "json") {
		return "", "", false
	}
	return format, path, true
}

func readBounded(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}
	defer f.Close()

	limited := io.LimitReader(f, plan.MaxPlanBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}
	if int64(len(data)) > plan.MaxPlanBytes {
		return nil, errors.New("plan exceeds maximum supported size")
	}
	return data, nil
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

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: testule validate [--format text|json] <plan.yaml>")
}
