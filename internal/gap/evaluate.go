package gap

import (
	"fmt"
	"sort"

	"github.com/hackelia-micrantha/testule/internal/evidence"
	"github.com/hackelia-micrantha/testule/internal/plan"
)

type observed struct {
	status string
	ref    string
}

func Evaluate(p *plan.TestPlan, evidences []*evidence.Evidence, subjectRevision string) (Report, []plan.Diagnostic) {
	fingerprint, err := plan.Fingerprint(p)
	if err != nil {
		return Report{}, []plan.Diagnostic{{Code: "fingerprint_error", Message: err.Error()}}
	}

	diagnostics := validateEvidenceIdentity(p, fingerprint, evidences, subjectRevision)
	if len(diagnostics) > 0 {
		return Report{}, diagnostics
	}

	observedByRequirement := make(map[string][]observed)
	for _, record := range evidences {
		for _, observation := range record.Observations {
			ref := record.Metadata.Name + "#" + observation.ID
			addObserved(observedByRequirement, "level", observation.Coverage.Levels, observation.Status, ref)
			addObserved(observedByRequirement, "behavior", observation.Coverage.Behaviors, observation.Status, ref)
			addObserved(observedByRequirement, "generation", observation.Coverage.Generation, observation.Status, ref)
		}
	}

	report := Report{
		Plan:            p.Metadata.Name,
		PlanFingerprint: fingerprint,
		Subject:         p.Subject.Component,
		SubjectRevision: subjectRevision,
		Complete:        true,
		Entries:         []Entry{},
	}

	for _, requirement := range plan.DeclaredRequirements(p) {
		entry := Entry{
			Dimension:   requirement.Dimension,
			Value:       requirement.Value,
			Disposition: requirement.Disposition,
			Rationale:   requirement.Rationale,
			Evidence:    []string{},
		}

		if requirement.Disposition == "inapplicable" {
			entry.State = StateInapplicable
		} else {
			observations := observedByRequirement[key(requirement.Dimension, requirement.Value)]
			entry.State, entry.Evidence = evaluateState(observations)
		}

		updateSummary(&report.Summary, entry.State)
		if entry.State == StateFailed {
			report.Complete = false
		}
		if requirement.Disposition == "required" && entry.State != StateSatisfied {
			report.Complete = false
		}
		report.Entries = append(report.Entries, entry)
	}

	return report, nil
}

func validateEvidenceIdentity(p *plan.TestPlan, fingerprint string, evidences []*evidence.Evidence, subjectRevision string) []plan.Diagnostic {
	var diagnostics []plan.Diagnostic
	seen := make(map[string]struct{}, len(evidences))
	for i, record := range evidences {
		prefix := fmt.Sprintf("evidence[%d]", i)
		if record == nil || record.Metadata == nil || record.Plan == nil || record.Subject == nil {
			diagnostics = append(diagnostics, plan.Diagnostic{Code: "invalid_evidence", Path: prefix, Message: "evidence must be decoded and validated before evaluation"})
			continue
		}

		name := record.Metadata.Name
		if _, duplicate := seen[name]; duplicate {
			diagnostics = append(diagnostics, plan.Diagnostic{Code: "duplicate_evidence", Path: prefix + ".metadata.name", Message: "evidence metadata.name must be unique across inputs"})
		}
		seen[name] = struct{}{}

		if record.Plan.Name != p.Metadata.Name {
			diagnostics = append(diagnostics, plan.Diagnostic{Code: "evidence_plan_mismatch", Path: prefix + ".plan.name", Message: "does not match evaluated plan"})
		}
		if record.Plan.Fingerprint != fingerprint {
			diagnostics = append(diagnostics, plan.Diagnostic{Code: "evidence_plan_fingerprint_mismatch", Path: prefix + ".plan.fingerprint", Message: "does not match evaluated plan fingerprint"})
		}
		if record.Subject.Component != p.Subject.Component {
			diagnostics = append(diagnostics, plan.Diagnostic{Code: "evidence_subject_mismatch", Path: prefix + ".subject.component", Message: "does not match evaluated subject"})
		}
		if record.Subject.Revision != subjectRevision {
			diagnostics = append(diagnostics, plan.Diagnostic{Code: "evidence_subject_revision_mismatch", Path: prefix + ".subject.revision", Message: "does not match requested subject revision"})
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

func addObserved(target map[string][]observed, dimension string, values []string, status, ref string) {
	for _, value := range values {
		k := key(dimension, value)
		target[k] = append(target[k], observed{status: status, ref: ref})
	}
}

func key(dimension, value string) string {
	return dimension + "\x00" + value
}

func evaluateState(observations []observed) (State, []string) {
	if len(observations) == 0 {
		return StateMissing, []string{}
	}

	refs := make([]string, 0, len(observations))
	seenRefs := make(map[string]struct{}, len(observations))
	hasPassed := false
	hasFailed := false
	hasSkipped := false
	hasUnsupported := false
	for _, observation := range observations {
		if _, ok := seenRefs[observation.ref]; !ok {
			refs = append(refs, observation.ref)
			seenRefs[observation.ref] = struct{}{}
		}
		switch observation.status {
		case "passed":
			hasPassed = true
		case "failed":
			hasFailed = true
		case "skipped":
			hasSkipped = true
		case "unsupported":
			hasUnsupported = true
		}
	}
	sort.Strings(refs)

	switch {
	case hasFailed:
		return StateFailed, refs
	case hasPassed:
		return StateSatisfied, refs
	case hasSkipped:
		return StateSkipped, refs
	case hasUnsupported:
		return StateUnsupported, refs
	default:
		return StateMissing, refs
	}
}

func updateSummary(summary *Summary, state State) {
	switch state {
	case StateSatisfied:
		summary.Satisfied++
	case StateMissing:
		summary.Missing++
	case StateUnsupported:
		summary.Unsupported++
	case StateSkipped:
		summary.Skipped++
	case StateFailed:
		summary.Failed++
	case StateInapplicable:
		summary.Inapplicable++
	}
}
