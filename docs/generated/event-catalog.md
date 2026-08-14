<!--
generated: true
source: internal/telemetry/schema.go (Attr* constants)
generator: scripts/generate-event-catalog.sh
generator_version: 1
do not edit manually — regenerate with: make docs-generate
-->

# Event / Attribute Catalog

OTel span and `slog` attribute keys currently emitted by Strategist,
extracted from `internal/telemetry/schema.go`. See
`docs/adr/0024-pluggable-governance-and-telemetry.md` for the proposed
(not yet built) `EventSink`/event-envelope design this catalog will
feed once that lands.

| Go Constant | Attribute Key |
|---|---|
| `AttrPhase` | `strategist.phase` |
| `AttrStatus` | `strategist.status` |
| `AttrComponent` | `strategist.component` |
| `AttrSkill` | `strategist.skill` |
| `AttrSelectedSkill` | `strategist.selected_skill` |
| `AttrArtifact` | `strategist.artifact` |
| `AttrArtifactPath` | `strategist.artifact.path` |
| `AttrReason` | `strategist.reason` |
| `AttrCacheHit` | `strategist.cache.hit` |
| `AttrTarget` | `strategist.target` |
| `AttrMandates` | `strategist.mandates.count` |
| `AttrMission` | `strategist.mission` |
| `AttrMissionID` | `strategist.mission_id` |
| `AttrCorrelationID` | `strategist.correlation_id` |
| `AttrRuntimeMode` | `strategist.runtime_mode` |
| `AttrOutputProfile` | `strategist.output_profile` |
| `AttrGateType` | `strategist.gate.type` |
| `AttrGateStatus` | `strategist.gate.status` |
| `AttrGateResponse` | `strategist.gate.response` |
| `AttrApprovalPolicy` | `strategist.approval_policy` |
| `AttrTransitionGroup` | `strategist.transition_group` |
| `AttrCheckpointPath` | `strategist.checkpoint.path` |
| `AttrStartToIntakeMS` | `strategist.metrics.t_start_to_intake_ms` |
| `AttrIntakeToRangerMS` | `strategist.metrics.t_intake_to_ranger_ms` |
| `AttrTotalWallTimeMS` | `strategist.metrics.total_wall_time_ms` |
| `AttrTokensIn` | `strategist.metrics.tokens_in` |
| `AttrTokensOut` | `strategist.metrics.tokens_out` |
| `AttrLinesEmitted` | `strategist.metrics.lines_emitted` |
| `AttrPipelineRoute` | `strategist.pipeline_route` |
| `AttrDecisionReason` | `strategist.decision_reason` |
| `AttrRole` | `strategist.role` |
| `AttrRoute` | `strategist.route` |
| `AttrRouteReason` | `strategist.route_reason` |
| `AttrRouteConfidence` | `strategist.route_confidence` |
| `AttrEvidenceState` | `strategist.evidence_state` |
| `AttrDiscoverySubtype` | `strategist.discovery_subtype` |
| `AttrProvider` | `strategist.provider` |
| `AttrIntakeToScoutMS` | `strategist.metrics.t_intake_to_scout_ms` |
| `AttrScoutToRangerMS` | `strategist.metrics.t_scout_to_ranger_ms` |
| `AttrRangerToArchivistMS` | `strategist.metrics.t_ranger_to_archivist_ms` |
| `AttrArchivistToGateMS` | `strategist.metrics.t_archivist_to_gate_ms` |
| `AttrGateWaitMS` | `strategist.metrics.t_gate_wait_ms` |
| `AttrGateToSniperMS` | `strategist.metrics.t_gate_to_sniper_ms` |
| `AttrSniperToDoneMS` | `strategist.metrics.t_sniper_to_done_ms` |
| `AttrDocumentationScope` | `strategist.documentation_scope` |
| `AttrHandoffChallengeStatus` | `strategist.handoff_challenge.status` |
| `AttrHandoffChallengeCriticalFailures` | `strategist.handoff_challenge.critical_failures` |
| `AttrHandoffChallengeTypes` | `strategist.handoff_challenge.types` |
| `AttrBasePath` | `strategist.base_path` |
| `AttrConflictCount` | `strategist.sniper.conflict_count` |
