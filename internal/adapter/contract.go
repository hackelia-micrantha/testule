package adapter

import (
	"context"

	"github.com/hackelia-micrantha/testule/internal/evidence"
)

// ProtocolVersion identifies the incubating semantic contract.
// It is intentionally not yet a stable public adapter transport or ABI.
const ProtocolVersion = "testule.adapter/v1alpha1"

type Capability string

const (
	CapabilityTestExecute    Capability = "test.execute"
	CapabilityEvidenceImport Capability = "evidence.import"
)

type TerminalStatus string

const (
	StatusCompleted            TerminalStatus = "completed"
	StatusDenied               TerminalStatus = "denied"
	StatusUnsupported          TerminalStatus = "unsupported"
	StatusTimedOut             TerminalStatus = "timed_out"
	StatusCancelled            TerminalStatus = "cancelled"
	StatusInfrastructureFailed TerminalStatus = "infrastructure_failed"
)

type Descriptor struct {
	ProtocolVersion string       `json:"protocolVersion"`
	ID              string       `json:"id"`
	Version         string       `json:"version"`
	Capabilities    []Capability `json:"capabilities"`
}

type Availability struct {
	Capability Capability `json:"capability"`
	Available  bool       `json:"available"`
	Reason     string     `json:"reason,omitempty"`
}

type ProbeResult struct {
	Adapter      string         `json:"adapter"`
	Availability []Availability `json:"availability"`
}

type PlanBinding struct {
	APIVersion  string `json:"apiVersion"`
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
	Component   string `json:"component"`
}

type Coverage struct {
	Level            string `json:"level,omitempty"`
	Behavior         string `json:"behavior,omitempty"`
	Generation       string `json:"generation,omitempty"`
	Visibility       string `json:"visibility,omitempty"`
	QualityAttribute string `json:"qualityAttribute,omitempty"`
}

// Invocation contains Testule-domain intent plus bounded adapter-owned input.
// It deliberately contains no executable/command/shell field. The full
// TestPlan is not sent to adapters; the core supplies the immutable binding
// needed to attribute normalized Evidence.
type Invocation struct {
	Capability      Capability        `json:"capability"`
	Plan            PlanBinding       `json:"plan"`
	SubjectRevision string            `json:"subjectRevision"`
	EnvironmentID   string            `json:"environmentId"`
	RunID           string            `json:"runId"`
	TargetID        string            `json:"targetId,omitempty"`
	Coverage        Coverage          `json:"coverage,omitempty"`
	AdapterOptions  map[string]string `json:"adapterOptions,omitempty"`
	Input           []byte            `json:"input,omitempty"`
}

type Result struct {
	Status      TerminalStatus     `json:"status"`
	Evidence    *evidence.Evidence `json:"evidence,omitempty"`
	Diagnostics []string           `json:"diagnostics,omitempty"`
}

type Target struct {
	ID    string `json:"id"`
	Label string `json:"label,omitempty"`
}

type Adapter interface {
	Describe() Descriptor
	Probe(context.Context) ProbeResult
	Invoke(context.Context, Invocation) Result
}

// Discoverer is optional. Importers and analyzers do not need to fabricate
// target inventories merely to satisfy the common adapter contract.
type Discoverer interface {
	Discover(context.Context) ([]Target, error)
}
