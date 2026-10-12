package jeweler

// EvidenceState is the provider-neutral state of one activation prerequisite.
// Static catalog or contract presence is never enough to produce certified
// probe or health evidence.
type EvidenceState string

const (
	// EvidenceCertified confirms that a readiness prerequisite was observed.
	EvidenceCertified EvidenceState = "certified"
	// EvidenceUnknown indicates that a readiness prerequisite was not observed.
	EvidenceUnknown EvidenceState = "unknown"
	// EvidenceStale indicates that readiness evidence is no longer current.
	EvidenceStale EvidenceState = "stale"
	// EvidenceFailed indicates that a readiness prerequisite failed.
	EvidenceFailed EvidenceState = "failed"
	// EvidenceUnsupported indicates that a prerequisite is not supported.
	EvidenceUnsupported EvidenceState = "unsupported"
	// EvidenceBlocked indicates that a prerequisite is explicitly blocked.
	EvidenceBlocked EvidenceState = "blocked"
	// EvidenceUnavailable indicates that a prerequisite could not be checked.
	EvidenceUnavailable EvidenceState = "unavailable"
	// EvidenceUnauthorized indicates that checking requires authorization.
	EvidenceUnauthorized EvidenceState = "unauthorized"
	// EvidenceTimeout indicates that a prerequisite check timed out.
	EvidenceTimeout EvidenceState = "timeout"
	// EvidenceMalformed indicates that collected evidence was malformed.
	EvidenceMalformed EvidenceState = "malformed"
	// EvidenceTeardownFailed indicates that provider teardown failed.
	EvidenceTeardownFailed EvidenceState = "teardown_failed"
)

// EvidenceVector keeps each readiness prerequisite independently observable.
// Package, identity, provenance and API are structural; delegate, probe and
// health must be certified before a provider can answer a consumer query.
type EvidenceVector struct {
	Package    EvidenceState `json:"package"`
	Identity   EvidenceState `json:"identity"`
	Provenance EvidenceState `json:"provenance"`
	API        EvidenceState `json:"api"`
	Delegate   EvidenceState `json:"delegate"`
	Probe      EvidenceState `json:"probe"`
	Health     EvidenceState `json:"health"`
}

// ReadinessFromEvidence maps the first unmet prerequisite to an actionable,
// fail-closed readiness state. Ordering is deliberate and stable for
// diagnostics and machine consumers.
func ReadinessFromEvidence(evidence EvidenceVector) Readiness {
	checks := []struct {
		name  string
		state EvidenceState
	}{
		{"package", evidence.Package},
		{"identity", evidence.Identity},
		{"provenance", evidence.Provenance},
		{"api", evidence.API},
		{"delegate", evidence.Delegate},
		{"probe", evidence.Probe},
		{"health", evidence.Health},
	}
	for _, check := range checks {
		if check.state == EvidenceCertified {
			continue
		}
		return Readiness{
			Status:   evidenceStatus(check.state),
			Reason:   check.name + "_" + readinessReason(check.state),
			Evidence: evidence,
		}
	}
	return Readiness{Status: "ready", Reason: "activation_evidence_verified", Evidence: evidence}
}

func evidenceStatus(state EvidenceState) string {
	if state == "" || state == EvidenceUnknown {
		return "unverified"
	}
	if state == EvidenceBlocked || state == EvidenceUnsupported || state == EvidenceUnauthorized {
		return "blocked"
	}
	return "degraded"
}

func readinessReason(state EvidenceState) string {
	if state == "" || state == EvidenceUnknown {
		return "unverified"
	}
	return string(state)
}
