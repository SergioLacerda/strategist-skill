package conformance

import "fmt"

// EvidenceAuthority identifies the authority that owns a provider binding.
type EvidenceAuthority string

const (
	// AuthorityCustomLock is the durable authority for Custom bindings.
	AuthorityCustomLock EvidenceAuthority = "custom_lock"
	// AuthorityRankedCatalog is the authority for Ranked certification inputs.
	AuthorityRankedCatalog EvidenceAuthority = "ranked_catalog"
)

// EvidenceProjection is a common view of evidence, not a new binding owner.
type EvidenceProjection struct {
	Authority  EvidenceAuthority
	Provider   string
	Provenance string
	State      EvidenceState
	Live       bool
}

// ProjectEvidence preserves authority and prevents static evidence from
// claiming live certification.
func ProjectEvidence(authority EvidenceAuthority, provider, provenance string, state EvidenceState, live bool) (EvidenceProjection, error) {
	if authority != AuthorityCustomLock && authority != AuthorityRankedCatalog {
		return EvidenceProjection{}, fmt.Errorf("evidence projection: unknown authority %q", authority)
	}
	if provider == "" || provenance == "" || !validState(state) {
		return EvidenceProjection{}, fmt.Errorf("evidence projection: provider, provenance, and state are required")
	}
	if state == StateCertified && !live {
		state = StateUnknown
	}
	return EvidenceProjection{Authority: authority, Provider: provider, Provenance: provenance, State: state, Live: live}, nil
}

// RejectAuthorityConflict fails closed without repairing a Custom lock or
// selecting a Ranked fallback.
func RejectAuthorityConflict(custom, ranked EvidenceProjection) error {
	if custom.Authority != AuthorityCustomLock || ranked.Authority != AuthorityRankedCatalog {
		return fmt.Errorf("evidence projection: expected Custom and Ranked authorities")
	}
	if custom.Provider != ranked.Provider {
		return fmt.Errorf("evidence projection: authority conflict")
	}
	return nil
}
