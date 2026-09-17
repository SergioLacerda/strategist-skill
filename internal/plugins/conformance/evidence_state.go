package conformance

import "github.com/SergioLacerda/strategist-skill/internal/domain"

// EvidenceState is the explicit operator-facing classification of conformance
// evidence. Unknown and unsupported are deliberately distinct from failed:
// neither claims that the provider failed a test nor silently promotes it.
type EvidenceState string

const (
	// EvidenceCertified means all required evidence is current and accepted.
	EvidenceCertified EvidenceState = "certified"
	// EvidenceStale means evidence was present but no longer matches inputs.
	EvidenceStale EvidenceState = "stale"
	// EvidenceFailed means evidence was evaluated and rejected.
	EvidenceFailed EvidenceState = "failed"
	// EvidenceUnknown means the required evidence could not be determined.
	EvidenceUnknown EvidenceState = "unknown"
	// EvidenceUnsupported means the connector cannot produce this evidence.
	EvidenceUnsupported EvidenceState = "unsupported"
)

// State returns the stable classification for a certification evaluation.
func (r CertificationResult) State() EvidenceState {
	if r.Accepted {
		return EvidenceCertified
	}
	for _, reason := range r.ReasonCodes {
		if reason == "certification_stale" {
			return EvidenceStale
		}
	}
	return EvidenceFailed
}

// StateForReadiness translates connector evidence without promoting static
// metadata to runtime certification. Blocked is a failed observation; unknown
// and unsupported remain distinct so callers can explain missing capability.
func StateForReadiness(status domain.ReadinessStatus) EvidenceState {
	switch status {
	case domain.ReadinessReady:
		return EvidenceCertified
	case domain.ReadinessUnsupported:
		return EvidenceUnsupported
	case domain.ReadinessUnknown:
		return EvidenceUnknown
	case domain.ReadinessBlocked:
		return EvidenceFailed
	default:
		return EvidenceUnknown
	}
}
