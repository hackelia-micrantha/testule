package plan

import (
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
		count := validateRequirements(p.Requirements, &diagnostics)
		if count == 0 {
			diagnostics = append(diagnostics, Diagnostic{Code: "no_requirements", Path: "requirements", Message: "at least one supported requirement is required"})
		}
	}

	sort.SliceStable(diagnostics, func(i, j int) bool {
		if diagnostics[i].Path != diagnostics[j].Path {
			return diagnostics[i].Path < diagnostics[j].Path
		}
		if diagnostics[i].Code != diagnostics[j].Code {
			return diagnostics[i].Code < diagnostics[j].Code
		}
		return diagnostics[i].Message < diagnostics[j].Message
	})
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
	for _, r := range name {
		if unicode.IsControl(r) {
			*diagnostics = append(*diagnostics, Diagnostic{Code: "control_character", Path: "metadata.name", Message: "name must not contain control characters"})
			break
		}
	}
}

func validateRequirements(r *Requirements, diagnostics *[]Diagnostic) int {
	count := 0
	if r.Levels != nil {
		entries := []struct {
			path  string
			value *string
		}{
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
		entries := []struct {
			path  string
			value *string
		}{
			{"requirements.behaviors.positive", r.Behaviors.Positive},
			{"requirements.behaviors.negative", r.Behaviors.Negative},
			{"requirements.behaviors.boundary", r.Behaviors.Boundary},
			{"requirements.behaviors.adversarial", r.Behaviors.Adversarial},
		}
		count += validateDispositions(entries, diagnostics)
	}
	return count
}

func validateDispositions(entries []struct {
	path  string
	value *string
}, diagnostics *[]Diagnostic) int {
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
