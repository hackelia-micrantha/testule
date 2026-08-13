package plan

import "sort"

func DeclaredRequirements(p *TestPlan) []Requirement {
	if p == nil || p.Requirements == nil {
		return nil
	}

	var requirements []Requirement
	add := func(dimension, value string, disposition *string) {
		if disposition == nil {
			return
		}
		requirements = append(requirements, Requirement{
			Dimension:   dimension,
			Value:       value,
			Disposition: *disposition,
		})
	}

	if levels := p.Requirements.Levels; levels != nil {
		add("level", "unit", levels.Unit)
		add("level", "component", levels.Component)
		add("level", "contract", levels.Contract)
		add("level", "integration", levels.Integration)
		add("level", "system", levels.System)
		add("level", "endToEnd", levels.EndToEnd)
	}
	if behaviors := p.Requirements.Behaviors; behaviors != nil {
		add("behavior", "positive", behaviors.Positive)
		add("behavior", "negative", behaviors.Negative)
		add("behavior", "boundary", behaviors.Boundary)
		add("behavior", "adversarial", behaviors.Adversarial)
	}
	if generation := p.Requirements.Generation; generation != nil {
		add("generation", "example", generation.Example)
		add("generation", "generated", generation.Generated)
		add("generation", "property", generation.Property)
		add("generation", "fuzz", generation.Fuzz)
		add("generation", "model", generation.Model)
		add("generation", "aiAssisted", generation.AIAssisted)
	}
	for _, omission := range p.Requirements.Inapplicable {
		requirements = append(requirements, Requirement{
			Dimension:   omission.Dimension,
			Value:       omission.Value,
			Disposition: "inapplicable",
			Rationale:   omission.Rationale,
		})
	}

	sort.Slice(requirements, func(i, j int) bool {
		if requirements[i].Dimension != requirements[j].Dimension {
			return requirements[i].Dimension < requirements[j].Dimension
		}
		if requirements[i].Value != requirements[j].Value {
			return requirements[i].Value < requirements[j].Value
		}
		return requirements[i].Disposition < requirements[j].Disposition
	})
	return requirements
}
