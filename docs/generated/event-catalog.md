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
| `AttrWeapon` | `strategist.weapon` |
| `AttrWeaponComponent` | `strategist.weapon.component` |
| `AttrWeaponParentInvocation` | `strategist.weapon.parent_invocation_id` |
| `AttrWeaponInvocationEvidence` | `strategist.weapon.invocation_evidence` |
| `AttrModel` | `strategist.model` |
| `AttrEffort` | `strategist.effort` |
| `AttrLevelSource` | `strategist.level_source` |
| `AttrAbility` | `strategist.ability` |
| `AttrInitiativeAdviceID` | `strategist.initiative.advice_id` |
| `AttrInitiativeMechanism` | `strategist.initiative.mechanism` |
| `AttrInitiativeMechanismLabel` | `strategist.initiative.mechanism_label` |
| `AttrInitiativePolicyVersion` | `strategist.initiative.policy_version` |
| `AttrInitiativePolicyDigest` | `strategist.initiative.policy_digest` |
| `AttrInitiativeTrigger` | `strategist.initiative.trigger` |
| `AttrInitiativeAlignment` | `strategist.initiative.alignment` |
| `AttrInitiativeConfidenceCeiling` | `strategist.initiative.confidence_ceiling` |
| `AttrInitiativeResultStatus` | `strategist.initiative.result_status` |
| `AttrInitiativeEvidenceRefs` | `strategist.initiative.evidence_refs` |
| `AttrInitiativeOutcomeIDs` | `strategist.initiative.outcome_ids` |
| `AttrInitiativeObservedModel` | `strategist.initiative.observed_model` |
| `AttrInitiativeObservedProvider` | `strategist.initiative.observed_provider` |
| `AttrInitiativeObservedEffort` | `strategist.initiative.observed_effort` |
| `AttrInitiativeObservedLevelSource` | `strategist.initiative.observed_level_source` |
| `AttrInitiativeRecommendedCapability` | `strategist.initiative.recommended_capability` |
| `AttrInitiativeRecommendedEffort` | `strategist.initiative.recommended_effort` |
| `AttrInitiativeAdviceReused` | `strategist.initiative.advice_reused` |
| `AttrInitiativeSupersedes` | `strategist.initiative.supersedes` |
| `AttrInitiativeDeviationIDs` | `strategist.initiative.deviation_ids` |
| `AttrInitiativeChallengeReasons` | `strategist.initiative.challenge_reasons` |
| `AttrInitiativeSourceRole` | `strategist.initiative.source_role` |
| `AttrInitiativeAssessmentID` | `strategist.initiative.assessment_id` |
| `AttrInitiativeInputDigest` | `strategist.initiative.input_digest` |
| `AttrInitiativeAlgorithmVersion` | `strategist.initiative.algorithm_version` |
| `AttrInitiativeCalibrationStatus` | `strategist.initiative.calibration_status` |
| `AttrInitiativeAssessed` | `strategist.initiative.assessed` |
| `AttrInitiativeEffective` | `strategist.initiative.effective` |
| `AttrInitiativeEvidenceExpected` | `strategist.initiative.evidence.expected` |
| `AttrInitiativeEvidenceVerified` | `strategist.initiative.evidence.verified` |
| `AttrInitiativeEvidenceSatisfied` | `strategist.initiative.evidence.satisfied` |
| `AttrInitiativeEvidencePartial` | `strategist.initiative.evidence.partial` |
| `AttrInitiativeEvidenceBlocked` | `strategist.initiative.evidence.blocked` |
| `AttrInitiativeEvidenceMissing` | `strategist.initiative.evidence.missing` |
| `AttrInitiativeEvidenceInvalid` | `strategist.initiative.evidence.invalid` |
| `AttrInitiativeEvidenceConflicting` | `strategist.initiative.evidence.conflicting` |
| `AttrInitiativeReasonCount` | `strategist.initiative.reason_count` |
| `AttrInitiativeEscalationRequested` | `strategist.initiative.escalation.requested` |
| `AttrInitiativeEscalationAccepted` | `strategist.initiative.escalation.accepted` |
| `AttrInitiativeEscalationOutcome` | `strategist.initiative.escalation.outcome` |
| `AttrInitiativeEscalationStatus` | `strategist.initiative.escalation.status` |
| `AttrInitiativeEscalationRequestID` | `strategist.initiative.escalation.request_id` |
| `AttrRoleRun` | `strategist.role_run` |
| `AttrConfidencePolicyVersion` | `strategist.confidence.policy_version` |
| `AttrConfidenceEventID` | `strategist.confidence.event_id` |
| `AttrConfidenceCorrelationKey` | `strategist.confidence.correlation_key` |
| `AttrClaimKind` | `strategist.confidence.claim_kind` |
| `AttrConfidenceLevel` | `strategist.confidence.level` |
| `AttrConfidencePercent` | `strategist.confidence.percent` |
| `AttrConfidenceEvidenceClass` | `strategist.confidence.evidence_class` |
| `AttrConfidenceEvidenceIDs` | `strategist.confidence.evidence_ids` |
| `AttrConfidenceEvidenceClasses` | `strategist.confidence.evidence_classes` |
| `AttrConfidenceSampleSize` | `strategist.confidence.sample_size` |
| `AttrConfidenceCalibrationStatus` | `strategist.confidence.calibration_status` |
| `AttrConfidenceGroundTruthRef` | `strategist.confidence.ground_truth_ref` |
| `AttrConfidenceGroundTruthKind` | `strategist.confidence.ground_truth_kind` |
| `AttrConfidenceGroundTruthOutcome` | `strategist.confidence.ground_truth_outcome` |
| `AttrConfidenceCoverageStatus` | `strategist.confidence.coverage_status` |
| `AttrConfidenceViolation` | `strategist.confidence.violation` |
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
| `AttrClaimMissionIDs` | `strategist.sniper.claim_mission_ids` |
