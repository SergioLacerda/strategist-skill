package i18n

// RuntimeMessages holds localized runtime strings emitted by Strategist agents during mission execution.
// These strings were previously embedded in persona YAML files under content_by_lang.
// The canonical English wording for each persona lives in strategist/personas/*.yaml;
// this bundle provides the Portuguese (pt-BR) localized alternative and a neutral English baseline.
type RuntimeMessages struct {
	// Intake phase
	IntakeSummary       string
	IntakeIndexModeNone string

	// Ranger (discovery) phase
	RangerStart string
	RangerDone  string

	// Archivist (refinement) phase
	ArchivistStart string
	ArchivistDone  string

	// Sniper (documentation materialization) phase
	SniperStart    string
	SniperTaskDone string
	SniperDone     string

	// Review gate (approval gate replacement)
	ReviewGatePrompt string

	// Opportunity attack
	OpportunityDetected string
	OpportunityGate     string
	OpportunitySignal   string

	// Quick draw route
	QuickDrawDetected string
	QuickDrawGate     string
	QuickDrawSuccess  string

	// Knowledge / chests
	TreasureChestFound string
	SideQuestDetected  string

	// Pipeline checkpoint
	MissionCheckpoint string

	// Compliance
	ComplianceSummary string

	// Execution tasks list
	ExecutionTasksHeader string
	ExecutionTaskLine    string

	// ADR stage
	AdrOpportunity string
	AdrGate        string

	// Mission close
	AnalysisDeliveredResult string
	ResponseComplete        string
	MissionComplete         string
	MissionMetrics          string

	// Rendering helpers
	PhaseTimelineEntry string
	ArtifactEntry      string
}

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

	ReviewGatePrompt: "🚦 **Gate [{mission_id}]:** AWAITING CONFIRMATION\n" +
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
		"To materialize documentation: re-invoke Strategist and accept the review gate.",

	ResponseComplete: "⚖️ **Compliance [{mission_id}]:** pipeline_compliant={pipeline_compliant} | phases={phases_run}",
	MissionComplete:  "[render mission_envelope.close — status_label: MISSION COMPLETE]",
	MissionMetrics: "[Strategist] metrics mission={mission_id} t_start_to_intake_ms={t_start_to_intake_ms}" +
		" t_intake_to_ranger_ms={t_intake_to_ranger_ms} total_wall_time_ms={total_wall_time_ms}" +
		" tokens_in={tokens_in} tokens_out={tokens_out} lines_emitted={lines_emitted}",

	PhaseTimelineEntry: "  {icon} {phase_label} → {result_label}",
	ArtifactEntry:      "  📁 {key}: {path}",
}

// PTBRRuntime is the Portuguese (pt-BR) runtime message bundle.
var PTBRRuntime = RuntimeMessages{ //nolint:dupl
	IntakeSummary: "Missão recebida: {task_type} | delivery={delivery_strategy} |" +
		" compatibility={legacy_compatibility} | urgency={urgency} | intent={execution_intent}",
	IntakeIndexModeNone: "⚠️ **Nota de governança:** `intake_index_mode: none` — nenhum contexto de governança" +
		" foi indexado para esta query. Status atual do execution_gate: {execution_gate}.",

	RangerStart: "🎯 **Ranger [{mission_id}]:** iniciando reconhecimento. skill={provider}",
	RangerDone: "🎯 **Ranger [{mission_id}]:** missão de reconhecimento concluída.\n" +
		"  ✶ channeling mana  ████████▓░░░░░░░░░░░░░░░░░░░  25% · Ranger ✓\n" +
		"Artefato em: {artifact_path}",

	ArchivistStart: "📚 **Archivist [{mission_id}]:** iniciando análise e refinamento. skill={provider}",
	ArchivistDone: "📚 **Archivist [{mission_id}]:** análise refinada.\n" +
		"  ✶ channeling mana  ████████████████▓░░░░░░░░░░░  50% · Archivist ✓\n" +
		"Artefatos em: {artifact_path}",

	SniperStart:    "🗡️ **Sniper [{mission_id}]:** alvo confirmado — iniciando materialização de documentação.",
	SniperTaskDone: "🗡️ **Sniper [{mission_id}]:** alvo {done}/{total} materializado — {task_title}",
	SniperDone: "🗡️ **Sniper [{mission_id}]:** materialização de documentação concluída.\n" +
		"  ✶ channeling mana  ████████████████████████████  100% ✓\n" +
		"Relatório em: {artifact_path}",

	ReviewGatePrompt: "🚦 **Gate [{mission_id}]:** AGUARDANDO CONFIRMAÇÃO\n" +
		"  ✶ channeling mana  ████████████████████████▓░░░  75% · aguardando revisão\n\n" +
		"Plano em: {artifact_path}\n\n" +
		"Revisar e confirmar? (sim / nao / revisar)",

	OpportunityDetected: "⚔️ **Ataque de Oportunidade** — {count} item(s) detectado(s)\n{items_brief}",
	OpportunityGate:     "⚔️ **Side Quests disponíveis:**\n{manifest}\n\nAprovar? (sim / nao / selecionar)",
	OpportunitySignal:   "⚔️ **Ataque de oportunidade!** {count} item(s) detectado(s) — detalhes no gate.",

	QuickDrawDetected: "⚔️ **Quick Draw** detectado. Side quest curta iniciada (Ranger -> Archivist -> Gate).",
	QuickDrawGate:     "🚦 **Gate Quick Draw**\n\nideia: {idea}\n\nadicionar ideia? (sim / nao)",
	QuickDrawSuccess: "⚔️ Quick Draw concluído.\n" +
		"sucesso: ideia adicionada em {destination_path}\n" +
		"total de ideias: {total_ideas}\n" +
		"ideias similares (mesmo tema): {similar_ideas}",

	TreasureChestFound: "🎁 **Baú do tesouro encontrado!** [{chest_id}] — {description}",
	SideQuestDetected:  "🗺️ **Side quest encontrada!** {description}",

	MissionCheckpoint: "**Checkpoint — {mission_id}**\n" +
		"{step_1_icon} 1 — Ranger\n" +
		"{step_2_icon} 2 — Archivist\n" +
		"{step_3_icon} 3 — Gate\n" +
		"{step_4_icon} 4 — Sniper",

	ComplianceSummary: "⚖️ **Compliance [{mission_id}]:** {status}",

	ExecutionTasksHeader: "🗡️ **Sniper — materializando {total} alvo(s) de documentação:**",
	ExecutionTaskLine:    "{status_icon} {index} — {task_title}",

	AdrOpportunity: "⚔️ **Ataque de Oportunidade → ADR**\n\n" +
		"Esta missão contém decisões arquiteturais que merecem registro.\n" +
		"Side quest: Archivist escreve ADR → Gate → Sniper arquiva.\n\n" +
		"Gerar ADR para \"{mission_id}\"? (sim / nao)",
	AdrGate: "📚 **Archivist — rascunho de ADR:**\n\n---\n{draft_content}\n---\n\n" +
		"🚦 **Gate ADR:** AGUARDANDO CONFIRMAÇÃO\n\nArquivar ADR? (sim / nao)",

	AnalysisDeliveredResult: "Missão [{mission_id}] encerrada — análise entregue.\n" +
		"Análise em: {artifact_path}\n" +
		"Para materializar documentação: re-invocar Strategist e aceitar o gate de revisão.",

	ResponseComplete: "⚖️ **Compliance [{mission_id}]:** pipeline_compliant={pipeline_compliant} | fases={phases_run}",
	MissionComplete:  "[renderizar mission_envelope.close — status_label: MISSÃO CONCLUÍDA]",
	MissionMetrics: "[Strategist] metrics mission={mission_id} t_start_to_intake_ms={t_start_to_intake_ms}" +
		" t_intake_to_ranger_ms={t_intake_to_ranger_ms} total_wall_time_ms={total_wall_time_ms}" +
		" tokens_in={tokens_in} tokens_out={tokens_out} lines_emitted={lines_emitted}",

	PhaseTimelineEntry: "  {icon} {phase_label} → {result_label}",
	ArtifactEntry:      "  📁 {key}: {path}",
}
