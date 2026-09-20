package telemetry

import "github.com/SergioLacerda/strategist-skill/internal/domain"

// Ground-truth subjects and their allowed labels.
const (
	GroundTruthSubjectRoute              = "route"
	GroundTruthSubjectHandoffApplication = "handoff_application"
	GroundTruthSubjectGateOutcome        = "gate_outcome"

	GateOutcomeAccepted          = "accepted"
	GateOutcomeRevisionRequested = "revision_requested"
	GateOutcomeRejected          = "rejected"

	RouteLabelConfirmed            = "confirmed"
	RouteLabelReversed             = "reversed"
	RouteLabelRiskUnderclassified  = "risk_underclassified"
	RouteLabelUserOverride         = "user_override"
	HandoffApplicationLabelApplied = "applied"
	HandoffApplicationLabelMissed  = "not_applied"
)

var groundTruthLabelsBySubject = map[string]map[string]bool{
	GroundTruthSubjectRoute: {
		RouteLabelConfirmed: true, RouteLabelReversed: true,
		RouteLabelRiskUnderclassified: true, RouteLabelUserOverride: true,
	},
	GroundTruthSubjectHandoffApplication: {
		HandoffApplicationLabelApplied: true, HandoffApplicationLabelMissed: true,
	},
	GroundTruthSubjectGateOutcome: {
		GateOutcomeAccepted: true, GateOutcomeRevisionRequested: true, GateOutcomeRejected: true,
	},
}

var groundTruthKinds = map[string]bool{
	domain.GroundTruthUserRevision: true,
	domain.GroundTruthHandoff:      true,
	domain.GroundTruthDownstream:   true,
}
