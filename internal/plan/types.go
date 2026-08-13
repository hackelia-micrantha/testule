package plan

type TestPlan struct {
	APIVersion   string        `yaml:"apiVersion"`
	Kind         string        `yaml:"kind"`
	Metadata     *Metadata     `yaml:"metadata"`
	Subject      *Subject      `yaml:"subject"`
	Requirements *Requirements `yaml:"requirements"`
}

type Metadata struct {
	Name string `yaml:"name"`
}

type Subject struct {
	Component string `yaml:"component"`
}

type Requirements struct {
	Levels    *Levels    `yaml:"levels,omitempty"`
	Behaviors *Behaviors `yaml:"behaviors,omitempty"`
}

type Levels struct {
	Unit        *string `yaml:"unit,omitempty"`
	Component   *string `yaml:"component,omitempty"`
	Contract    *string `yaml:"contract,omitempty"`
	Integration *string `yaml:"integration,omitempty"`
	System      *string `yaml:"system,omitempty"`
	EndToEnd    *string `yaml:"endToEnd,omitempty"`
}

type Behaviors struct {
	Positive    *string `yaml:"positive,omitempty"`
	Negative    *string `yaml:"negative,omitempty"`
	Boundary    *string `yaml:"boundary,omitempty"`
	Adversarial *string `yaml:"adversarial,omitempty"`
}
