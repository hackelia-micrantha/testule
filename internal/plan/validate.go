package plan

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	APIVersion = "testule.dev/v1alpha1"
	Kind       = "TestPlan"
)

var allowedDisposition = map[string]struct{}{
	"required": {},
	"optional": {},
}

var dimensionValues = map[string]map[string]struct{}{
	"level": {
		"unit": {}, "component": {}, "contract": {}, "integration": {}, "system": {}, "endToEnd": {},
	},
	"behavior": {
		"positive": {}, "negative": {}, "boundary": {}, "adversarial": {},
	},
	"generation": {
		"example": {}, "generated": {}, "property": {}, "fuzz": {}, "model": {}, "aiAssisted": {},
	},
}

func Validate(p *TestPlan) []Diagnostic {
	var diagnostics []Diagnostic

	if p.APIVersion != APIVersion {
		diagnostics = append(diagnostics, Diagnostic{Code: "invalid_api_version", Path: "apiVersion", Message: "must be exactly " + APIVersion})
	}
	if p.Kind != Kind {
		diagnostics = append(diagnostics, Diagnostic{Code: "invalid_kind", Path: "kind", Message: "must be exactly " + Kind})
	}

	if p.Metadata == nil {
		diagnostics = append(diagnostics, Diagnostic{Code: "required", Path: "metadata", Message: "metadata is required"})
	} else {
		validateName(p.Metadata.Name, &diagnostics)
	}

	if p.Subject == nil {
		diagnostics = append(diagnostics, Diagnostic{Code: "required", Path: "subject", Message: "subject is required"})
	} else if strings.TrimSpace(p.Subject.Component) == "" {
		diagnostics = append(diagnostics, Diagnostic{Code: "required", Path: "subject.component", Message: "component is required"})
	}

	if p.Requirements == nil {
		diagnostics = append(diagnostics, Diagnostic{Code: "required", Path: "requirements", Message: "requirements is required"})
	} else {
		count := validateRequirements(p, &diagnostics)
		if count == 0 {
			diagnostics = append(diagnostics, Diagnostic{Code: "no_requirements", Path: "requirements", Message: "at least one supported required or optional requirement is required"})
		}
	}

	sortDiagnostics(diagnostics)
	return diagnostics
}

func validateName(name string, diagnostics *[]Diagnostic) {
	if strings.TrimSpace(name) == "" {
		*diagnostics = append(*diagnostics, Diagnostic{Code: "required", Path: "metadata.name", Message: "name is required"})
		return
	}
	if utf8.RuneCountInString(name) > 128 {
		*diagnostics = append(*diagnostics, Diagnostic{Code: "too_long", Path: "metadata.name", Message: "name must be at most 128 Unicode code points"})
	}
	if containsControl(name) {
		*diagnostics = append(*diagnostics, Diagnostic{Code: "control_character", Path: "metadata.name", Message: "name must not contain control characters"})
	}
}

func validateRequirements(p *TestPlan, diagnostics *[]Diagnostic) int {
	r := p.Requirements
	count := 0
	if r.Levels != nil {
		entries := []dispositionEntry{
			{"requirements.levels.unit", r.Levels.Unit},
			{"requirements.levels.component", r.Levels.Component},
			{"requirements.levels.contract", r.Levels.Contract},
			{"requirements.levels.integration", r.Levels.Integration},
			{"requirements.levels.system", r.Levels.System},
			{"requirements.levels.endToEnd", r.Levels.EndToEnd},
		}
		count += validateDispositions(entries, diagnostics)
	}
	if r.Behaviors != nil {
		entries := []dispositionEntry{
			{"requirements.behaviors.positive", r.Behaviors.Positive},
			{"requirements.behaviors.negative", r.Behaviors.Negative},
			{"requirements.behaviors.boundary", r.Behaviors.Boundary},
			{"requirements.behaviors.adversarial", r.Behaviors.Adversarial},
		}
		count += validateDispositions(entries, diagnostics)
	}
	if r.Generation != nil {
		entries := []dispositionEntry{
			{"requirements.generation.example", r.Generation.Example},
			{"requirements.generation.generated", r.Generation.Generated},
			{"requirements.generation.property", r.Generation.Property},
			{"requirements.generation.fuzz", r.Generation.Fuzz},
			{"requirements.generation.model", r.Generation.Model},
			{"requirements.generation.aiAssisted", r.Generation.AIAssisted},
		}
		count += validateDispositions(entries, diagnostics)
	}
	validateInapplicable(p, diagnostics)
	return count
}

type dispositionEntry struct {
	path  string
	value *string
}

func validateDispositions(entries []dispositionEntry, diagnostics *[]Diagnostic) int {
	count := 0
	for _, entry := range entries {
		if entry.value == nil {
			continue
		}
		count++
		if _, ok := allowedDisposition[*entry.value]; !ok {
			*diagnostics = append(*diagnostics, Diagnostic{
				Code:    "invalid_value",
				Path:    entry.path,
				Message: "must be one of: required, optional",
			})
		}
	}
	return count
}

func validateInapplicable(p *TestPlan, diagnostics *[]Diagnostic) {
	omissions := p.Requirements.Inapplicable
	if len(omissions) > 64 {
		*diagnostics = append(*diagnostics, Diagnostic{
			Code:    "too_many_items",
			Path:    "requirements.inapplicable",
			Message: "must contain at most 64 entries",
		})
	}

	seen := make(map[string]struct{}, len(omissions))
	for i, omission := range omissions {
		path := fmt.Sprintf("requirements.inapplicable[%d]", i)
		values, dimensionOK := dimensionValues[omission.Dimension]
		if !dimensionOK {
			*diagnostics = append(*diagnostics, Diagnostic{
				Code:    "invalid_value",
				Path:    path + ".dimension",
				Message: "must be one of: level, behavior, generation",
			})
		} else if _, valueOK := values[omission.Value]; !valueOK {
			*diagnostics = append(*diagnostics, Diagnostic{
				Code:    "invalid_value",
				Path:    path + ".value",
				Message: "is not supported for dimension " + omission.Dimension,
			})
		}

		rationale := strings.TrimSpace(omission.Rationale)
		if rationale == "" {
			*diagnostics = append(*diagnostics, Diagnostic{
				Code:    "required",
				Path:    path + ".rationale",
				Message: "rationale is required for an inapplicable requirement",
			})
		} else {
			if utf8.RuneCountInString(omission.Rationale) > 1024 {
				*diagnostics = append(*diagnostics, Diagnostic{
					Code:    "too_long",
					Path:    path + ".rationale",
					Message: "rationale must be at most 1024 Unicode code points",
				})
			}
			if containsControl(omission.Rationale) {
				*diagnostics = append(*diagnostics, Diagnostic{
					Code:    "control_character",
					Path:    path + ".rationale",
					Message: "rationale must not contain control characters",
				})
			}
		}

		key := omission.Dimension + "\x00" + omission.Value
		if _, duplicate := seen[key]; duplicate {
			*diagnostics = append(*diagnostics, Diagnostic{
				Code:    "duplicate",
				Path:    path,
				Message: "duplicate inapplicable requirement",
			})
		}
		seen[key] = struct{}{}

		if dimensionOK {
			if _, valueOK := values[omission.Value]; valueOK && declaredWithoutInapplicable(p, omission.Dimension, omission.Value) {
				*diagnostics = append(*diagnostics, Diagnostic{
					Code:    "contradiction",
					Path:    path,
					Message: "cannot mark a declared required or optional requirement as inapplicable",
				})
			}
		}
	}
}

func declaredWithoutInapplicable(p *TestPlan, dimension, value string) bool {
	r := p.Requirements
	switch dimension {
	case "level":
		if r.Levels == nil {
			return false
		}
		values := map[string]*string{
			"unit": r.Levels.Unit, "component": r.Levels.Component, "contract": r.Levels.Contract,
			"integration": r.Levels.Integration, "system": r.Levels.System, "endToEnd": r.Levels.EndToEnd,
		}
		return values[value] != nil
	case "behavior":
		if r.Behaviors == nil {
			return false
		}
		values := map[string]*string{
			"positive": r.Behaviors.Positive, "negative": r.Behaviors.Negative,
			"boundary": r.Behaviors.Boundary, "adversarial": r.Behaviors.Adversarial,
		}
		return values[value] != nil
	case "generation":
		if r.Generation == nil {
			return false
		}
		values := map[string]*string{
			"example": r.Generation.Example, "generated": r.Generation.Generated, "property": r.Generation.Property,
			"fuzz": r.Generation.Fuzz, "model": r.Generation.Model, "aiAssisted": r.Generation.AIAssisted,
		}
		return values[value] != nil
	default:
		return false
	}
}

func containsControl(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func sortDiagnostics(diagnostics []Diagnostic) {
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
