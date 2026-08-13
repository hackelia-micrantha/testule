package capability

import "encoding/json"

const (
	ProtocolVersion  = "testule.capability/v1alpha1"
	MaxEnvelopeBytes = 1 << 20
)

type Status string

const (
	StatusSucceeded            Status = "succeeded"
	StatusFailed               Status = "failed"
	StatusDenied               Status = "denied"
	StatusUnsupported          Status = "unsupported"
	StatusTimedOut             Status = "timed_out"
	StatusPartial              Status = "partial"
	StatusInfrastructureFailed Status = "infrastructure_failed"
)

type ArtifactTrust string

const (
	ArtifactProposed              ArtifactTrust = "proposed"
	ArtifactStructurallyValidated ArtifactTrust = "structurally_validated"
	ArtifactAnalysisValidated     ArtifactTrust = "analysis_validated"
	ArtifactExecutionValidated    ArtifactTrust = "execution_validated"
)

type CapabilityRef struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Subject struct {
	ID       string `json:"id"`
	Revision string `json:"revision"`
}

type FilesystemRequirements struct {
	Read  []string `json:"read,omitempty"`
	Write []string `json:"write,omitempty"`
}

type RequestedAuthority struct {
	Filesystem    FilesystemRequirements `json:"filesystem,omitempty"`
	Process       bool                   `json:"process,omitempty"`
	Network       bool                   `json:"network,omitempty"`
	Secrets       bool                   `json:"secrets,omitempty"`
	Devices       bool                   `json:"devices,omitempty"`
	MaxWallTimeMS int64                  `json:"maxWallTimeMs,omitempty"`
	MaxMemoryBytes int64                 `json:"maxMemoryBytes,omitempty"`
	MaxOutputBytes int64                 `json:"maxOutputBytes,omitempty"`
}

type Invocation struct {
	ProtocolVersion   string             `json:"protocolVersion"`
	ID                string             `json:"id"`
	Capability        CapabilityRef      `json:"capability"`
	Subject           Subject            `json:"subject"`
	Input             json.RawMessage    `json:"input,omitempty"`
	RequestedAuthority RequestedAuthority `json:"requestedAuthority,omitempty"`
}

type Diagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Artifact struct {
	ID    string        `json:"id"`
	Kind  string        `json:"kind"`
	Ref   string        `json:"ref"`
	Trust ArtifactTrust `json:"trust"`
}

type ProviderRef struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Result struct {
	ProtocolVersion string        `json:"protocolVersion"`
	InvocationID    string        `json:"invocationId"`
	Capability      CapabilityRef `json:"capability"`
	Subject         Subject       `json:"subject"`
	Status          Status        `json:"status"`
	Diagnostics     []Diagnostic  `json:"diagnostics"`
	EvidenceRefs    []string      `json:"evidenceRefs,omitempty"`
	Artifacts       []Artifact    `json:"artifacts,omitempty"`
	Provider        ProviderRef   `json:"provider"`
}

type HostContext struct {
	ActorID string
	RunID   string
}
