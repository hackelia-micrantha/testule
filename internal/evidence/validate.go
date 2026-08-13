package evidence

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/hackelia-micrantha/testule/internal/plan"
)

const Kind = "Evidence"

var allowedStatus = map[string]struct{}{
	"passed":      {},
	"failed":      {},
	"skipped":     {},
	"unsupported": {},
}

var allowedCoverage = map[string]map[string]struct{}{
	"levels": {
		"unit": {}, "component": {}, "contract": {}, "integration": {}, "system": {}, "endToEnd": {},
	},
	"behaviors": {
		"positive": {}, "negative": {}, "boundary": {}, "adversarial": {},
	},
	"generation": {
		"example": {}, "generated": {}, "property": {}, "fuzz": {}, "model": {}, "aiAssisted": {},
	},
	"visibility": {
		"blackBox": {}, "grayBox": {}, "whiteBox": {},
	},
	"qualityAttributes": {
		"functional": {}, "security": {}, "performance": {}, "resilience": {}, "compatibility": {}, "operational": {},
	},
}

func Validate(e *Evidence) []plan.Diagnostic {
	var diagnostics []plan.Diagnostic

	if e.APIVersion != plan.APIVersion {
		diagnostics = append(diagnostics, plan.Diagnostic{Code: "invalid_api_version", Path: "apiVersion", Message: "must be exactly " + plan.APIVersion})
	}
	if e.Kind != Kind {
		diagnostics = append(diagnostics, plan.Diagnostic{Code: "invalid_kind", Path: "kind", Message: "must be exactly " + Kind})
	}

	if e.Metadata == nil {
		diagnostics = append(diagnostics, plan.Diagnostic{Code: "required", Path: "metadata", Message: "metadata is required"})
	} else {
		validateBoundedString(e.Metadata.Name, "metadata.name", 128, true, &diagnostics)
	}
	if e.Plan == nil {
		diagnostics = append(diagnostics, plan.Diagnostic{Code: "required", Path: "plan", Message: "plan reference is required"})
	} else {
		validateBoundedString(e.Plan.Name, "plan.name", 128, true, &diagnostics)
		validateFingerprint(e.Plan.Fingerprint, &diagnostics)
	}
	if e.Subject == nil {
		diagnostics = append(diagnostics, plan.Diagnostic{Code: "required", Path: "subject", Message: "subject is required"})
	} else {
		validateBoundedString(e.Subject.Component, "subject.component", 256, true, &diagnostics)
		validateBoundedString(e.Subject.Revision, "subject.revision", 256, true, &diagnostics)
	}
	if e.Environment == nil {
		diagnostics = append(diagnostics, plan.Diagnostic{Code: "required", Path: "environment", Message: "environment is required"})
	} else {
		validateBoundedString(e.Environment.ID, "environment.id", 256, true, &diagnostics)
	}
	if e.Provenance == nil {
		diagnostics = append(diagnostics, plan.Diagnostic{Code: "required", Path: "provenance", Message: "provenance is required"})
	} else {
		validateProvenance(e.Provenance, &diagnostics)
	}

	validateObservations(e.Observations, &diagnostics)
	sortDiagnostics(diagnostics)
	return diagnostics
}

func validateFingerprint(value string, diagnostics *[]plan.Diagnostic) {
	if value == "" {
		*diagnostics = append(*diagnostics, plan.Diagnostic{Code: "required", Path: "plan.fingerprint", Message: "fingerprint is required"})
		return
	}
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		*diagnostics = append(*diagnostics, plan.Diagnostic{Code: "invalid_value", Path: "plan.fingerprint", Message: "must be sha256:<64 lowercase hex characters>"})
		return
	}
	for _, r := range strings.TrimPrefix(value, "sha256:") {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			*diagnostics = append(*diagnostics, plan.Diagnostic{Code: "invalid_value", Path: "plan.fingerprint", Message: "must be sha256:<64 lowercase hex characters>"})
			return
		}
	}
}

func validateProvenance(p *Provenance, diagnostics *[]plan.Diagnostic) {
	validateBoundedString(p.Producer, "provenance.producer", 256, true, diagnostics)
	validateBoundedString(p.RunID, "provenance.runId", 256, true, diagnostics)
	if len(p.References) > 32 {
		*diagnostics = append(*diagnostics, plan.Diagnostic{
			Code:    "too_many_items",
			Path:    "provenance.references",
			Message: "must contain at most 32 references",
		})
	}
	seen := make(map[string]struct{}, len(p.References))
	for i, reference := range p.References {
		path := fmt.Sprintf("provenance.references[%d]", i)
		validateBoundedString(reference, path, 2048, true, diagnostics)
		if _, ok := seen[reference]; ok {
			*diagnostics = append(*diagnostics, plan.Diagnostic{Code: "duplicate", Path: path, Message: "duplicate provenance reference"})
		}
		seen[reference] = struct{}{}
	}
}

func validateObservations(observations []Observation, diagnostics *[]plan.Diagnostic) {
	if len(observations) == 0 {
		*diagnostics = append(*diagnostics, plan.Diagnostic{Code: "required", Path: "observations", Message: "at least one observation is required"})
		return
	}
	if len(observations) > 256 {
		*diagnostics = append(*diagnostics, plan.Diagnostic{Code: "too_many_items", Path: "observations", Message: "must contain at most 256 observations"})
	}

	seenIDs := make(map[string]struct{}, len(observations))
	for i, observation := range observations {
		path := fmt.Sprintf("observations[%d]", i)
		validateBoundedString(observation.ID, path+".id", 128, true, diagnostics)
		if _, ok := seenIDs[observation.ID]; ok {
			*diagnostics = append(*diagnostics, plan.Diagnostic{Code: "duplicate", Path: path + ".id", Message: "observation id must be unique within an evidence record"})
		}
		seenIDs[observation.ID] = struct{}{}

		if _, ok := allowedStatus[observation.Status]; !ok {
			*diagnostics = append(*diagnostics, plan.Diagnostic{
				Code:    "invalid_value",
				Path:    path + ".status",
				Message: "must be one of: passed, failed, skipped, unsupported",
			})
		}

		coverageCount := 0
		coverageCount += validateCoverageValues(observation.Coverage.Levels, "levels", path+".coverage.levels", diagnostics)
		coverageCount += validateCoverageValues(observation.Coverage.Behaviors, "behaviors", path+".coverage.behaviors", diagnostics)
		coverageCount += validateCoverageValues(observation.Coverage.Generation, "generation", path+".coverage.generation", diagnostics)
		coverageCount += validateCoverageValues(observation.Coverage.Visibility, "visibility", path+".coverage.visibility", diagnostics)
		coverageCount += validateCoverageValues(observation.Coverage.QualityAttributes, "qualityAttributes", path+".coverage.qualityAttributes", diagnostics)
		if coverageCount == 0 {
			*diagnostics = append(*diagnostics, plan.Diagnostic{
				Code:    "required",
				Path:    path + ".coverage",
				Message: "at least one supported coverage value is required",
			})
		}
	}
}

func validateCoverageValues(values []string, dimension, path string, diagnostics *[]plan.Diagnostic) int {
	if len(values) > 32 {
		*diagnostics = append(*diagnostics, plan.Diagnostic{Code: "too_many_items", Path: path, Message: "must contain at most 32 values"})
	}
	allowed := allowedCoverage[dimension]
	seen := make(map[string]struct{}, len(values))
	for i, value := range values {
		itemPath := fmt.Sprintf("%s[%d]", path, i)
		if _, ok := allowed[value]; !ok {
			*diagnostics = append(*diagnostics, plan.Diagnostic{Code: "invalid_value", Path: itemPath, Message: "unsupported coverage value"})
		}
		if _, ok := seen[value]; ok {
			*diagnostics = append(*diagnostics, plan.Diagnostic{Code: "duplicate", Path: itemPath, Message: "duplicate coverage value"})
		}
		seen[value] = struct{}{}
	}
	return len(values)
}

func validateBoundedString(value, path string, maxRunes int, required bool, diagnostics *[]plan.Diagnostic) {
	if required && strings.TrimSpace(value) == "" {
		*diagnostics = append(*diagnostics, plan.Diagnostic{Code: "required", Path: path, Message: "value is required"})
		return
	}
	if utf8.RuneCountInString(value) > maxRunes {
		*diagnostics = append(*diagnostics, plan.Diagnostic{Code: "too_long", Path: path, Message: fmt.Sprintf("must be at most %d Unicode code points", maxRunes)})
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			*diagnostics = append(*diagnostics, plan.Diagnostic{Code: "control_character", Path: path, Message: "must not contain control characters"})
			break
		}
	}
}

func sortDiagnostics(diagnostics []plan.Diagnostic) {
	sort.SliceStable(diagnostics, func(i, j int) bool {
		if diagnostics[i].Path != diagnostics[j].Path {
			return diagnostics[i].Path < diagnostics[j].Path
		}
		if diagnostics[i].Code != diagnostics[j].Code {
			return diagnostics[i].Code < diagnostics[j].Code
		}
		return diagnostics[i].Message < diagnostics[j].Message
	})
}
