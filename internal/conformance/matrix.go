// Package conformance provides provider-neutral client conformance matrices.
package conformance

// EvidenceTier separates deterministic structural evidence from live-provider
// evidence. Structural rows never imply that a provider was invoked.
type EvidenceTier string

const (
	// EvidenceStructural is evidence derived from deterministic local data.
	EvidenceStructural EvidenceTier = "structural"
	// EvidenceLive is evidence collected from an explicit provider probe.
	EvidenceLive EvidenceTier = "live"
)

// EvidenceState is the client-facing classification expected from a row.
type EvidenceState string

const (
	// StateCertified indicates that the required evidence was verified.
	StateCertified EvidenceState = "certified"
	// StateStale indicates that previously certified evidence is outdated.
	StateStale EvidenceState = "stale"
	// StateFailed indicates that evidence collection or validation failed.
	StateFailed EvidenceState = "failed"
	// StateUnknown indicates that evidence has not been classified.
	StateUnknown EvidenceState = "unknown"
	// StateUnsupported indicates that the surface cannot provide the evidence.
	StateUnsupported EvidenceState = "unsupported"
	// StateBlocked indicates that policy prevented evidence collection.
	StateBlocked EvidenceState = "blocked"
)

// Client describes one release-supported client or adapter surface.
type Client struct {
	ID                string   `yaml:"id" json:"id"`
	Owner             string   `yaml:"owner" json:"owner"`
	Surface           string   `yaml:"surface" json:"surface"`
	Roles             []string `yaml:"roles" json:"roles"`
	ProviderModes     []string `yaml:"provider_modes" json:"provider_modes"`
	UnsupportedPolicy string   `yaml:"unsupported_policy" json:"unsupported_policy"`
}

// Row is one provider-neutral conformance case.
type Row struct {
	ID              string        `yaml:"id" json:"id"`
	Client          string        `yaml:"client" json:"client"`
	Role            string        `yaml:"role" json:"role"`
	Slot            string        `yaml:"slot" json:"slot"`
	ProviderMode    string        `yaml:"provider_mode" json:"provider_mode"`
	EnvelopeVersion string        `yaml:"envelope_version" json:"envelope_version"`
	EvidenceTier    EvidenceTier  `yaml:"evidence_tier" json:"evidence_tier"`
	ExpectedState   EvidenceState `yaml:"expected_state" json:"expected_state"`
	ReasonCode      string        `yaml:"reason_code" json:"reason_code"`
	AuthorityOwner  string        `yaml:"authority_owner" json:"authority_owner"`
	ContextFixture  string        `yaml:"context_fixture,omitempty" json:"context_fixture,omitempty"`
}

// Matrix is the versioned source for structural conformance reporting.
type Matrix struct {
	SchemaVersion string   `yaml:"schema_version" json:"schema_version"`
	Clients       []Client `yaml:"clients" json:"clients"`
	Rows          []Row    `yaml:"rows" json:"rows"`
}

// RowResult reports one evaluated matrix row.
type RowResult struct {
	RowID  string `json:"row_id"`
	Client string `json:"client"`
	Passed bool   `json:"passed"`
	Reason string `json:"reason,omitempty"`
}

// Report is deterministic for a given matrix and availability inventory.
type Report struct {
	SchemaVersion string      `json:"schema_version"`
	MatrixDigest  string      `json:"matrix_digest"`
	ClientCount   int         `json:"client_count"`
	RowCount      int         `json:"row_count"`
	Results       []RowResult `json:"results"`
}
