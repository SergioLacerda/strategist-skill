package i18n

// ENRuntime is the English runtime message bundle.
// Template variables use {key} placeholders matching persona content_by_lang.en conventions.
var ENRuntime = RuntimeMessages{ //nolint:dupl
	IntakeSummary: "Mission received: {task_type} | delivery={delivery_strategy} |" +
		" compatibility={legacy_compatibility} | urgency={urgency} | intent={execution_intent}",
	IntakeIndexModeNone: "⚠️ **Governance note:** `intake_index_mode: none` — no governance context was indexed" +
		" for this query. Current execution_gate: {execution_gate}.",

	RoleLevelHeader: "Fase: {phase_index}/04\n{role_name}\n{model_effort}",

	RoleStart: "{role_level_header}\n{role_emoji} **{role_title} [{mission_id}]:** {start_text}",
	RoleDone: "{role_level_header}\n{role_emoji} **{role_title} [{mission_id}]:** {done_text}\n" +
		"{artifact_label} {artifact_path}",
	RoleTaskDone: "{role_level_header}\n{role_emoji} **{role_title} [{mission_id}]:** {task_text}",
	RolePhrases: map[string]RolePhrase{
		"_default":  {Emoji: "🎭", StartText: "starting. skill={provider}", DoneText: "complete.", ArtifactLabel: "Artifact at:"},
		"ranger":    {Emoji: "🎯", StartText: "starting reconnaissance. skill={provider}", DoneText: "reconnaissance complete.", ArtifactLabel: "Artifact at:"},
		"archivist": {Emoji: "📚", StartText: "starting analysis and refinement. skill={provider}", DoneText: "analysis refined.", ArtifactLabel: "Artifacts at:"},
		"sniper":    {Emoji: "🗡️", StartText: "documentation target confirmed — starting materialization.", DoneText: "documentation materialization complete.", ArtifactLabel: "Report at:", TaskText: "target {done}/{total} materialized — {task_title}"},
	},

	ApprovalGatePrompt: "{role_level_header}\n🚦 **Gate [{mission_id}]:** AWAITING CONFIRMATION\n\n" +
		"Plan at: {artifact_path}\n\n" +
		"Review and confirm? (yes / no / review)",

	OpportunityDetected: "⚔️ **Opportunity Attack** — {count} item(s) detected\n{items_brief}",
	OpportunityGate:     "⚔️ **Available Side Quests:**\n{manifest}\n\nApprove? (yes / no / select)",
	OpportunitySignal:   "⚔️ **Opportunity Attack!** {count} item(s) detected — details at gate.",

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
