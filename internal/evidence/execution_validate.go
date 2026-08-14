package evidence

import (
	"fmt"
	pathpkg "path"
	"strings"

	"github.com/hackelia-micrantha/testule/internal/plan"
)

func ValidateExecutionArtifacts(record *Evidence) []plan.Diagnostic {
	var diagnostics []plan.Diagnostic
	if record == nil {
		return []plan.Diagnostic{{Code: "required", Path: "evidence", Message: "evidence is required"}}
	}
	validateExecution(record.Execution, &diagnostics)
	validateArtifacts(record.Artifacts, &diagnostics)
	sortDiagnostics(diagnostics)
	return diagnostics
}

func validateExecution(execution *Execution, diagnostics *[]plan.Diagnostic) {
	if execution == nil {
		return
	}
	validateBoundedString(execution.Adapter, "execution.adapter", 128, true, diagnostics)
	validateBoundedString(execution.Operation, "execution.operation", 64, true, diagnostics)
	validateBoundedString(execution.Tool, "execution.tool", 128, true, diagnostics)
	validateBoundedString(execution.ToolVersion, "execution.toolVersion", 256, true, diagnostics)
	validateBoundedString(execution.Package, "execution.package", 512, false, diagnostics)
	validateBoundedString(execution.Target, "execution.target", 256, false, diagnostics)
	if len(execution.Command) == 0 {
		*diagnostics = append(*diagnostics, plan.Diagnostic{Code: "required", Path: "execution.command", Message: "at least one command argument is required"})
	}
	if len(execution.Command) > 64 {
		*diagnostics = append(*diagnostics, plan.Diagnostic{Code: "too_many_items", Path: "execution.command", Message: "must contain at most 64 arguments"})
	}
	for i, argument := range execution.Command {
		validateBoundedString(argument, fmt.Sprintf("execution.command[%d]", i), 1024, true, diagnostics)
	}
	if execution.ExitCode < -1 || execution.ExitCode > 255 {
		*diagnostics = append(*diagnostics, plan.Diagnostic{Code: "invalid_value", Path: "execution.exitCode", Message: "must be between -1 and 255"})
	}
	if execution.DurationMillis < 0 {
		*diagnostics = append(*diagnostics, plan.Diagnostic{Code: "invalid_value", Path: "execution.durationMillis", Message: "must not be negative"})
	}
}

func validateArtifacts(artifacts []Artifact, diagnostics *[]plan.Diagnostic) {
	if len(artifacts) > 64 {
		*diagnostics = append(*diagnostics, plan.Diagnostic{Code: "too_many_items", Path: "artifacts", Message: "must contain at most 64 artifacts"})
	}
	seenNames := make(map[string]struct{}, len(artifacts))
	seenPaths := make(map[string]struct{}, len(artifacts))
	for i, artifact := range artifacts {
		base := fmt.Sprintf("artifacts[%d]", i)
		validateBoundedString(artifact.Name, base+".name", 256, true, diagnostics)
		if strings.ContainsAny(artifact.Name, `/\\`) || artifact.Name == "." || artifact.Name == ".." {
			*diagnostics = append(*diagnostics, plan.Diagnostic{Code: "invalid_value", Path: base + ".name", Message: "artifact name must not contain path separators"})
		}
		if _, exists := seenNames[artifact.Name]; exists {
			*diagnostics = append(*diagnostics, plan.Diagnostic{Code: "duplicate", Path: base + ".name", Message: "artifact name must be unique"})
		}
		seenNames[artifact.Name] = struct{}{}

		validateBoundedString(artifact.Role, base+".role", 128, true, diagnostics)
		validateArtifactPath(artifact.Path, base+".path", diagnostics)
		if _, exists := seenPaths[artifact.Path]; exists {
			*diagnostics = append(*diagnostics, plan.Diagnostic{Code: "duplicate", Path: base + ".path", Message: "artifact path must be unique"})
		}
		seenPaths[artifact.Path] = struct{}{}
		validateSHA256(artifact.SHA256, base+".sha256", diagnostics)
		validateBoundedString(artifact.MediaType, base+".mediaType", 256, false, diagnostics)
	}
}

func validateArtifactPath(value, diagnosticPath string, diagnostics *[]plan.Diagnostic) {
	validateBoundedString(value, diagnosticPath, 1024, true, diagnostics)
	if value == "" {
		return
	}
	if strings.Contains(value, `\\`) || strings.HasPrefix(value, "/") || pathpkg.Clean(value) != value || value == "." || value == ".." || strings.HasPrefix(value, "../") {
		*diagnostics = append(*diagnostics, plan.Diagnostic{Code: "invalid_value", Path: diagnosticPath, Message: "artifact path must be a clean relative slash-separated path"})
	}
}

func validateSHA256(value, diagnosticPath string, diagnostics *[]plan.Diagnostic) {
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		*diagnostics = append(*diagnostics, plan.Diagnostic{Code: "invalid_value", Path: diagnosticPath, Message: "must be sha256:<64 lowercase hex characters>"})
		return
	}
	for _, r := range strings.TrimPrefix(value, "sha256:") {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			*diagnostics = append(*diagnostics, plan.Diagnostic{Code: "invalid_value", Path: diagnosticPath, Message: "must be sha256:<64 lowercase hex characters>"})
			return
		}
	}
}
