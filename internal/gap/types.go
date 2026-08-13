package gap

type State string

const (
	StateSatisfied    State = "satisfied"
	StateMissing      State = "missing"
	StateUnsupported  State = "unsupported"
	StateSkipped      State = "skipped"
	StateFailed       State = "failed"
	StateInapplicable State = "inapplicable"
)

type Entry struct {
	Dimension   string   `json:"dimension"`
	Value       string   `json:"value"`
	Disposition string   `json:"disposition"`
	State       State    `json:"state"`
	Rationale   string   `json:"rationale,omitempty"`
	Evidence    []string `json:"evidence"`
}

type Summary struct {
	Satisfied    int `json:"satisfied"`
	Missing      int `json:"missing"`
	Unsupported  int `json:"unsupported"`
	Skipped      int `json:"skipped"`
	Failed       int `json:"failed"`
	Inapplicable int `json:"inapplicable"`
}

type Report struct {
	Plan            string  `json:"plan"`
	PlanFingerprint string  `json:"planFingerprint"`
	Subject         string  `json:"subject"`
	SubjectRevision string  `json:"subjectRevision"`
	Complete        bool    `json:"complete"`
	Entries         []Entry `json:"entries"`
	Summary         Summary `json:"summary"`
}
