package plan

type TestPlan struct {
	APIVersion   string        `yaml:"apiVersion" json:"apiVersion"`
	Kind         string        `yaml:"kind" json:"kind"`
	Metadata     *Metadata     `yaml:"metadata" json:"metadata"`
	Subject      *Subject      `yaml:"subject" json:"subject"`
	Requirements *Requirements `yaml:"requirements" json:"requirements"`
}

type Metadata struct {
	Name string `yaml:"name" json:"name"`
}

type Subject struct {
	Component string `yaml:"component" json:"component"`
}

type Requirements struct {
	Levels       *Levels        `yaml:"levels,omitempty" json:"levels,omitempty"`
	Behaviors    *Behaviors     `yaml:"behaviors,omitempty" json:"behaviors,omitempty"`
	Generation   *Generation    `yaml:"generation,omitempty" json:"generation,omitempty"`
	Inapplicable []Inapplicable `yaml:"inapplicable,omitempty" json:"inapplicable,omitempty"`
}

type Levels struct {
	Unit        *string `yaml:"unit,omitempty" json:"unit,omitempty"`
	Component   *string `yaml:"component,omitempty" json:"component,omitempty"`
	Contract    *string `yaml:"contract,omitempty" json:"contract,omitempty"`
	Integration *string `yaml:"integration,omitempty" json:"integration,omitempty"`
	System      *string `yaml:"system,omitempty" json:"system,omitempty"`
	EndToEnd    *string `yaml:"endToEnd,omitempty" json:"endToEnd,omitempty"`
}

type Behaviors struct {
	Positive    *string `yaml:"positive,omitempty" json:"positive,omitempty"`
	Negative    *string `yaml:"negative,omitempty" json:"negative,omitempty"`
	Boundary    *string `yaml:"boundary,omitempty" json:"boundary,omitempty"`
	Adversarial *string `yaml:"adversarial,omitempty" json:"adversarial,omitempty"`
}

type Generation struct {
	Example    *string `yaml:"example,omitempty" json:"example,omitempty"`
	Generated  *string `yaml:"generated,omitempty" json:"generated,omitempty"`
	Property   *string `yaml:"property,omitempty" json:"property,omitempty"`
	Fuzz       *string `yaml:"fuzz,omitempty" json:"fuzz,omitempty"`
	Model      *string `yaml:"model,omitempty" json:"model,omitempty"`
	AIAssisted *string `yaml:"aiAssisted,omitempty" json:"aiAssisted,omitempty"`
}

type Inapplicable struct {
	Dimension string `yaml:"dimension" json:"dimension"`
	Value     string `yaml:"value" json:"value"`
	Rationale string `yaml:"rationale" json:"rationale"`
}

type Requirement struct {
	Dimension   string `json:"dimension"`
	Value       string `json:"value"`
	Disposition string `json:"disposition"`
	Rationale   string `json:"rationale,omitempty"`
}
