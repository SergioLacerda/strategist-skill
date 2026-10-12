// Package hardening contains shared, machine-readable safety outcomes.
package hardening

import "fmt"

// Status is the closed vocabulary used by hardening checks.
type Status string

const (
	// StatusVerified means all required evidence matched.
	StatusVerified Status = "verified"
	// StatusUnknown means evidence was insufficient to classify the result.
	StatusUnknown Status = "unknown"
	// StatusUnavailable means the source could not be inspected.
	StatusUnavailable Status = "unavailable"
	// StatusDegraded means compatibility evidence is incomplete.
	StatusDegraded Status = "degraded"
	// StatusFailed means the check executed and failed.
	StatusFailed Status = "failed"
	// StatusStale means the source is older or divergent.
	StatusStale Status = "stale"
	// StatusBlocked means safety policy prevented continuation.
	StatusBlocked Status = "blocked"
)

// Outcome is an auditable result shared by integrity and restore checks.
type Outcome struct {
	Status      Status `json:"status"`
	Reason      string `json:"reason"`
	Remediation string `json:"remediation,omitempty"`
	Source      string `json:"source,omitempty"`
	MissionID   string `json:"mission_id,omitempty"`
	Phase       string `json:"phase,omitempty"`
	Digest      string `json:"digest,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

// Success reports whether the evidence is strong enough to be trusted.
func (o Outcome) Success() bool { return o.Status == StatusVerified }

// Authority identifies the owner of one protected structural fact.
type Authority struct {
	Fact          string `json:"fact"`
	Owner         string `json:"owner"`
	Authoritative bool   `json:"authoritative"`
}

// AuthorityMap is the complete set of protected fact owners.
type AuthorityMap []Authority

// Validate rejects duplicate facts and entries without an owner.
func (m AuthorityMap) Validate() error {
	seen := make(map[string]struct{}, len(m))
	for _, entry := range m {
		if entry.Fact == "" || entry.Owner == "" {
			return fmt.Errorf("authority map: fact and owner are required")
		}
		if _, exists := seen[entry.Fact]; exists {
			return fmt.Errorf("authority map: duplicate fact %q", entry.Fact)
		}
		seen[entry.Fact] = struct{}{}
	}
	return nil
}

// DefaultAuthorityMap describes the repository's existing ownership model.
func DefaultAuthorityMap() AuthorityMap {
	return AuthorityMap{
		{Fact: "runtime_config", Owner: ".strategist/active.yaml", Authoritative: true},
		{Fact: "custom_binding", Owner: ".strategist/plugins.lock", Authoritative: true},
		{Fact: "ranked_binding", Owner: "build_catalog_certification", Authoritative: true},
		{Fact: "mission_transition", Owner: "domain.MissionEngine", Authoritative: true},
		{Fact: "context_identity", Owner: "domain.ContextMaterializer", Authoritative: true},
		{Fact: "observability", Owner: "telemetry.Event", Authoritative: false},
	}
}
