package telemetry

// Attribute key constants for Strategist OTel spans and log records.
const (
	AttrPhase            = "strategist.phase"
	AttrStatus           = "strategist.status"
	AttrSkill            = "strategist.skill"
	AttrArtifact         = "strategist.artifact"
	AttrReason           = "strategist.reason"
	AttrCacheHit         = "strategist.cache.hit"
	AttrTarget           = "strategist.target"
	AttrMandates         = "strategist.mandates.count"
	AttrMission          = "strategist.mission"
	AttrStartToIntakeMS  = "strategist.metrics.t_start_to_intake_ms"
	AttrIntakeToRangerMS = "strategist.metrics.t_intake_to_ranger_ms"
	AttrTotalWallTimeMS  = "strategist.metrics.total_wall_time_ms"
	AttrTokensIn         = "strategist.metrics.tokens_in"
	AttrTokensOut        = "strategist.metrics.tokens_out"
	AttrLinesEmitted     = "strategist.metrics.lines_emitted"
)
