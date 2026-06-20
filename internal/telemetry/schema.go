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
)

const redactedPath = "<redacted-path>"

// SanitizePath replaces absolute paths with a sentinel before use as a span attribute.
// Call this on any string that may originate from user filesystem input before
// attaching it to a trace span.
func SanitizePath(p string) string {
	if filepath.IsAbs(p) {
		return redactedPath
	}
	return p
}
