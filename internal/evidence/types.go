package evidence

type Evidence struct {
	APIVersion   string         `yaml:"apiVersion" json:"apiVersion"`
	Kind         string         `yaml:"kind" json:"kind"`
	Metadata     *Metadata      `yaml:"metadata" json:"metadata"`
	Plan         *PlanReference `yaml:"plan" json:"plan"`
	Subject      *Subject       `yaml:"subject" json:"subject"`
	Environment  *Environment   `yaml:"environment" json:"environment"`
	Provenance   *Provenance    `yaml:"provenance" json:"provenance"`
	Execution    *Execution     `yaml:"execution,omitempty" json:"execution,omitempty"`
	Artifacts    []Artifact     `yaml:"artifacts,omitempty" json:"artifacts,omitempty"`
	Observations []Observation  `yaml:"observations" json:"observations"`
}

type Metadata struct {
	Name string `yaml:"name" json:"name"`
}

type PlanReference struct {
	Name        string `yaml:"name" json:"name"`
	Fingerprint string `yaml:"fingerprint" json:"fingerprint"`
}

type Subject struct {
	Component string `yaml:"component" json:"component"`
	Revision  string `yaml:"revision" json:"revision"`
}

type Environment struct {
	ID string `yaml:"id" json:"id"`
}

type Provenance struct {
	Producer   string   `yaml:"producer" json:"producer"`
	RunID      string   `yaml:"runId" json:"runId"`
	References []string `yaml:"references,omitempty" json:"references,omitempty"`
}

type Execution struct {
	Adapter         string   `yaml:"adapter" json:"adapter"`
	Operation       string   `yaml:"operation" json:"operation"`
	Tool            string   `yaml:"tool" json:"tool"`
	ToolVersion     string   `yaml:"toolVersion" json:"toolVersion"`
	Scope           string   `yaml:"scope,omitempty" json:"scope,omitempty"`
	Target          string   `yaml:"target,omitempty" json:"target,omitempty"`
	Command         []string `yaml:"command" json:"command"`
	ExitCode        int      `yaml:"exitCode" json:"exitCode"`
	DurationMillis  int64    `yaml:"durationMillis" json:"durationMillis"`
	TimedOut        bool     `yaml:"timedOut,omitempty" json:"timedOut,omitempty"`
	OutputTruncated bool     `yaml:"outputTruncated,omitempty" json:"outputTruncated,omitempty"`
}

type Artifact struct {
	Name      string `yaml:"name" json:"name"`
	Role      string `yaml:"role" json:"role"`
	Path      string `yaml:"path" json:"path"`
	SHA256    string `yaml:"sha256" json:"sha256"`
	MediaType string `yaml:"mediaType,omitempty" json:"mediaType,omitempty"`
}

type Observation struct {
	ID       string   `yaml:"id" json:"id"`
	Status   string   `yaml:"status" json:"status"`
	Coverage Coverage `yaml:"coverage" json:"coverage"`
}

type Coverage struct {
	Levels            []string `yaml:"levels,omitempty" json:"levels,omitempty"`
	Behaviors         []string `yaml:"behaviors,omitempty" json:"behaviors,omitempty"`
	Generation        []string `yaml:"generation,omitempty" json:"generation,omitempty"`
	Visibility        []string `yaml:"visibility,omitempty" json:"visibility,omitempty"`
	QualityAttributes []string `yaml:"qualityAttributes,omitempty" json:"qualityAttributes,omitempty"`
}
