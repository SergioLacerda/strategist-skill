package conformance

// BoundaryStage identifies the ordered ownership stage being evaluated.
type BoundaryStage string

const (
	// StageInRepo represents the in-repo staging boundary stage.
	StageInRepo BoundaryStage = "in_repo_staging"
	// StageExternalAdapter represents the external adapter boundary stage.
	StageExternalAdapter BoundaryStage = "external_adapter"
)

// BoundaryStatus is a fail-closed activation result for the selected stage.
type BoundaryStatus string

const (
	// BoundaryReady indicates that activation evidence is verified and ready.
	BoundaryReady BoundaryStatus = "ready"
	// BoundaryDegraded indicates that the boundary is operating in a degraded state.
	BoundaryDegraded BoundaryStatus = "degraded"
	// BoundaryBlocked indicates that the boundary is blocked due to policy or missing requirements.
	BoundaryBlocked BoundaryStatus = "blocked"
	// BoundaryUnverified indicates that the boundary activation status is unverified.
	BoundaryUnverified BoundaryStatus = "unverified"
)

// Ownership separates the Strategist host envelope from Treasure Chest domain ownership.
type Ownership struct {
	Host  string
	Skill string
}

// EvidenceVector retains each activation prerequisite independently.
type EvidenceVector struct {
	Package    EvidenceState
	Identity   EvidenceState
	Provenance EvidenceState
	API        EvidenceState
	Delegate   EvidenceState
	Probe      EvidenceState
	Health     EvidenceState
}

// TreasureChestBoundary is a projection, never a binding authority.
type TreasureChestBoundary struct {
	Stage     BoundaryStage
	Ownership Ownership
	Evidence  EvidenceVector
}

// BoundaryResult describes why a stage did or did not become invocable.
type BoundaryResult struct {
	Status              BoundaryStatus
	Reason              string
	InvocationReady     bool
	LegacyDataPreserved bool
}

// EvaluateTreasureChestBoundary makes activation depend on evidence for the selected stage.
func EvaluateTreasureChestBoundary(boundary TreasureChestBoundary) BoundaryResult {
	if result, invalid := validateBoundary(boundary); invalid {
		return result
	}
	return evaluateActivationEvidence(boundary.Evidence)
}

func validateBoundary(boundary TreasureChestBoundary) (BoundaryResult, bool) {
	if boundary.Ownership.Host == "" || boundary.Ownership.Skill == "" || boundary.Ownership.Host == boundary.Ownership.Skill {
		return boundaryResult(BoundaryBlocked, "ownership_conflict"), true
	}
	if boundary.Stage != StageInRepo && boundary.Stage != StageExternalAdapter {
		return boundaryResult(BoundaryBlocked, "stage_unsupported"), true
	}
	return BoundaryResult{}, false
}

func evaluateActivationEvidence(evidence EvidenceVector) BoundaryResult {
	if result, ok := staticEvidenceResult(evidence); ok {
		return result
	}
	return evaluateDelegateEvidence(evidence)
}

func evaluateDelegateEvidence(evidence EvidenceVector) BoundaryResult {
	if evidence.Delegate == StateCertified {
		return evaluateProbeAndHealthEvidence(evidence)
	}
	if evidence.Delegate == StateUnknown {
		return boundaryResult(BoundaryDegraded, "delegate_unwired")
	}
	return evidenceResult("delegate", evidence.Delegate, "delegate_unwired")
}

func evaluateProbeAndHealthEvidence(evidence EvidenceVector) BoundaryResult {
	if evidence.Probe != StateCertified {
		return evidenceResult("probe", evidence.Probe, "probe_unverified")
	}
	if evidence.Health != StateCertified {
		return evidenceResult("health", evidence.Health, "health_unverified")
	}
	return BoundaryResult{Status: BoundaryReady, Reason: "activation_evidence_verified", InvocationReady: true}
}

func staticEvidenceResult(evidence EvidenceVector) (BoundaryResult, bool) {
	for _, item := range []struct {
		name  string
		state EvidenceState
	}{
		{"package", evidence.Package}, {"identity", evidence.Identity}, {"provenance", evidence.Provenance}, {"api", evidence.API},
	} {
		if item.state != StateCertified {
			return evidenceResult(item.name, item.state, item.name+"_unverified"), true
		}
	}
	return BoundaryResult{}, false
}

func evidenceResult(name string, state EvidenceState, unknownReason string) BoundaryResult {
	if state == StateUnknown {
		return boundaryResult(BoundaryUnverified, unknownReason)
	}
	if state == StateBlocked || state == StateUnsupported || state == StateUnauthorized {
		return boundaryResult(BoundaryBlocked, name+"_"+string(state))
	}
	return boundaryResult(BoundaryDegraded, name+"_"+string(state))
}

func boundaryResult(status BoundaryStatus, reason string) BoundaryResult {
	return BoundaryResult{Status: status, Reason: reason, LegacyDataPreserved: true}
}
