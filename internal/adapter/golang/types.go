package golang

import (
	"time"

	"github.com/hackelia-micrantha/testule/internal/evidence"
	"github.com/hackelia-micrantha/testule/internal/plan"
)

const (
	AdapterID      = "go-native/v1alpha1"
	MaxOutputBytes = int64(1 << 20)
)

type Operation string

const (
	OperationTest   Operation = "test"
	OperationFuzz   Operation = "fuzz"
	OperationReplay Operation = "replay"
)

type Coverage struct {
	Level            string
	Behavior         string
	Generation       string
	Visibility       string
	QualityAttribute string
}

type RunConfig struct {
	Operation       Operation
	Plan            *plan.TestPlan
	SubjectRevision string
	Workspace       string
	Package         string
	Target          string
	EnvironmentID   string
	RunID           string
	Timeout         time.Duration
	FuzzTime        time.Duration
	Coverage        Coverage
}

type ReplayConfig struct {
	SourceEvidence  *evidence.Evidence
	EvidencePath    string
	SubjectRevision string
	Workspace       string
	EnvironmentID   string
	RunID           string
	Timeout         time.Duration
}

type PromoteConfig struct {
	SourceEvidence  *evidence.Evidence
	EvidencePath    string
	SubjectRevision string
	Workspace       string
}

type Result struct {
	Evidence     *evidence.Evidence
	EvidencePath string
	ArtifactDir  string
	Status       string
}
