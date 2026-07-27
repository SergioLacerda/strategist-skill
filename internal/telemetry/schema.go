package telemetry

import "path/filepath"

// Attribute key constants for Strategist OTel spans and log records.
const (
	AttrPhase            = "strategist.phase"
	AttrStatus           = "strategist.status"
	AttrComponent        = "strategist.component"
	AttrSkill            = "strategist.skill"
	AttrSelectedSkill    = "strategist.selected_skill"
	AttrArtifact         = "strategist.artifact"
	AttrArtifactPath     = "strategist.artifact.path"
	AttrReason           = "strategist.reason"
	AttrCacheHit         = "strategist.cache.hit"
	AttrTarget           = "strategist.target"
	AttrMandates         = "strategist.mandates.count"
	AttrMission          = "strategist.mission"
	AttrMissionID        = "strategist.mission_id"
	AttrCorrelationID    = "strategist.correlation_id"
	AttrRuntimeMode      = "strategist.runtime_mode"
	AttrOutputProfile    = "strategist.output_profile"
	AttrGateType         = "strategist.gate.type"
	AttrGateStatus       = "strategist.gate.status"
	AttrGateResponse     = "strategist.gate.response"
	AttrApprovalPolicy   = "strategist.approval_policy"
	AttrTransitionGroup  = "strategist.transition_group"
	AttrCheckpointPath   = "strategist.checkpoint.path"
	AttrStartToIntakeMS  = "strategist.metrics.t_start_to_intake_ms"
	AttrIntakeToRangerMS = "strategist.metrics.t_intake_to_ranger_ms"
	AttrTotalWallTimeMS  = "strategist.metrics.total_wall_time_ms"
	AttrTokensIn         = "strategist.metrics.tokens_in"
	AttrTokensOut        = "strategist.metrics.tokens_out"
	AttrLinesEmitted     = "strategist.metrics.lines_emitted"

	// AttrPipelineRoute and AttrDecisionReason are the canonical attribute names for
	// route-decision telemetry (mission route, e.g. "main", and a short
	// machine-readable reason for that route/readiness verdict). Emitted today only
	// from `strategist check --simulate`, the sole Go-side call site that produces a
	// route-shaped verdict; full mission routing (critical-hit) is a
	// prompt-time decision made by the LLM runtime from the narrative/machine
	// contracts, not the compiled CLI, and is not instrumented here.
	AttrPipelineRoute  = "strategist.pipeline_route"
	AttrDecisionReason = "strategist.decision_reason"

	// Scout route-decision attributes (see contracts/narrative/10-telemetry.md §
	// Scout Event and contracts/machine/scout-routing.yaml). These follow the same
	// documentation-parity pattern as AttrPipelineRoute/AttrDecisionReason above:
	// Scout's actual mission-time classification is a prompt-time decision made by
	// the LLM runtime from the narrative/machine contracts, not the compiled CLI.
	AttrRole             = "strategist.role"
	AttrRoute            = "strategist.route"
	AttrRouteReason      = "strategist.route_reason"
	AttrRouteConfidence  = "strategist.route_confidence"
	AttrEvidenceState    = "strategist.evidence_state"
	AttrDiscoverySubtype = "strategist.discovery_subtype"
	AttrProvider         = "strategist.provider"

	AttrIntakeToScoutMS     = "strategist.metrics.t_intake_to_scout_ms"
	AttrScoutToRangerMS     = "strategist.metrics.t_scout_to_ranger_ms"
	AttrRangerToArchivistMS = "strategist.metrics.t_ranger_to_archivist_ms"
	AttrArchivistToGateMS   = "strategist.metrics.t_archivist_to_gate_ms"
	AttrGateWaitMS          = "strategist.metrics.t_gate_wait_ms"
	AttrGateToSniperMS      = "strategist.metrics.t_gate_to_sniper_ms"
	AttrSniperToDoneMS      = "strategist.metrics.t_sniper_to_done_ms"
	AttrDocumentationScope  = "strategist.documentation_scope"

	// AttrBasePath, AttrConflictCount are the F3 revisit tripwire signal attrs
	// (ADR-0008 § F3 revisit tripwire, docs/adr/0008-single-session-assumption.md).
	AttrBasePath      = "strategist.base_path"
	AttrConflictCount = "strategist.sniper.conflict_count"
)

const redactedPath = "<redacted-path>"

const documentationScopeApprovedTargets = "approved_targets"

// SanitizePath replaces absolute paths with a sentinel before use as a span attribute.
// Call this on any string that may originate from user filesystem input before
// attaching it to a trace span.
func SanitizePath(p string) string {
	if filepath.IsAbs(p) {
		return redactedPath
	}
	return p
}
