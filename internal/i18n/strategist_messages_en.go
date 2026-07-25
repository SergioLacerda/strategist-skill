package i18n

// ENRuntime is the English runtime message bundle.
// Template variables use {key} placeholders matching persona content_by_lang.en conventions.
var ENRuntime = RuntimeMessages{ //nolint:dupl
	IntakeSummary: "Mission received: {task_type} | delivery={delivery_strategy} |" +
		" compatibility={legacy_compatibility} | urgency={urgency} | intent={execution_intent}",
	IntakeIndexModeNone: "⚠️ **Governance note:** `intake_index_mode: none` — no governance context was indexed" +
		" for this query. Current execution_gate: {execution_gate}.",

	RangerStart: "🎯 **Ranger [{mission_id}]:** starting reconnaissance. skill={provider}",
	RangerDone: "🎯 **Ranger [{mission_id}]:** reconnaissance complete.\n" +
		"  ✶ channeling mana  ████████▓░░░░░░░░░░░░░░░░░░░  25% · Ranger ✓\n" +
		"Artifact at: {artifact_path}",

	ArchivistStart: "📚 **Archivist [{mission_id}]:** starting analysis and refinement. skill={provider}",
	ArchivistDone: "📚 **Archivist [{mission_id}]:** analysis refined.\n" +
		"  ✶ channeling mana  ████████████████▓░░░░░░░░░░░  50% · Archivist ✓\n" +
		"Artifacts at: {artifact_path}",

	SniperStart:    "🗡️ **Sniper [{mission_id}]:** documentation target confirmed — starting materialization.",
	SniperTaskDone: "🗡️ **Sniper [{mission_id}]:** target {done}/{total} materialized — {task_title}",
	SniperDone: "🗡️ **Sniper [{mission_id}]:** documentation materialization complete.\n" +
		"  ✶ channeling mana  ████████████████████████████  100% ✓\n" +
		"Report at: {artifact_path}",

	ApprovalGatePrompt: "🚦 **Gate [{mission_id}]:** AWAITING CONFIRMATION\n" +
		"  ✶ channeling mana  ████████████████████████▓░░░  75% · awaiting review\n\n" +
		"Plan at: {artifact_path}\n\n" +
		"Review and confirm? (yes / no / review)",

	OpportunityDetected: "⚔️ **Opportunity Attack** — {count} item(s) detected\n{items_brief}",
	OpportunityGate:     "⚔️ **Available Side Quests:**\n{manifest}\n\nApprove? (yes / no / select)",
	OpportunitySignal:   "⚔️ **Opportunity Attack!** {count} item(s) detected — details at gate.",

	QuickDrawDetected: "⚔️ **Quick Draw** detected. Short side quest started (Ranger -> Archivist -> Gate).",
	QuickDrawGate:     "🚦 **Quick Draw Gate**\n\nidea: {idea}\n\nadd idea? (yes / no)",
	QuickDrawSuccess: "⚔️ Quick Draw complete.\n" +
		"success: idea added at {destination_path}\n" +
		"total ideas: {total_ideas}\n" +
		"similar ideas (same theme): {similar_ideas}",

	TreasureChestFound: "🎁 **Treasure chest found!** [{chest_id}] — {description}",
	SideQuestDetected:  "🗺️ **Side quest detected!** {description}",

	MissionCheckpoint: "**Checkpoint — {mission_id}**\n" +
		"{step_1_icon} 1 — Ranger\n" +
		"{step_2_icon} 2 — Archivist\n" +
		"{step_3_icon} 3 — Gate\n" +
		"{step_4_icon} 4 — Sniper",

	ComplianceSummary: "⚖️ **Compliance [{mission_id}]:** {status}",

	ExecutionTasksHeader: "🗡️ **Sniper — materializing {total} documentation target(s):**",
	ExecutionTaskLine:    "{status_icon} {index} — {task_title}",

	AdrOpportunity: "⚔️ **Opportunity Attack → ADR**\n\n" +
		"This mission contains architectural decisions worth recording.\n" +
		"Side quest: Archivist writes ADR → Gate → Sniper archives.\n\n" +
		"Generate ADR for \"{mission_id}\"? (yes / no)",
	AdrGate: "📚 **Archivist — ADR draft:**\n\n---\n{draft_content}\n---\n\n" +
		"🚦 **ADR Gate:** AWAITING CONFIRMATION\n\nArchive ADR? (yes / no)",

	AnalysisDeliveredResult: "Mission [{mission_id}] closed — analysis delivered.\n" +
		"Analysis at: {artifact_path}\n" +
		"To materialize documentation: re-invoke Strategist and accept the approval gate.",

	ResponseComplete: "⚖️ **Compliance [{mission_id}]:** pipeline_compliant={pipeline_compliant} | phases={phases_run}",
	MissionComplete:  "[render mission_envelope.close — status_label: MISSION COMPLETE]",
	MissionMetrics: "[Strategist] metrics mission={mission_id} t_start_to_intake_ms={t_start_to_intake_ms}" +
		" t_intake_to_ranger_ms={t_intake_to_ranger_ms} total_wall_time_ms={total_wall_time_ms}" +
		" tokens_in={tokens_in} tokens_out={tokens_out} lines_emitted={lines_emitted}",

	PhaseTimelineEntry: "  {icon} {phase_label} → {result_label}",
	ArtifactEntry:      "  📁 {key}: {path}",
}
