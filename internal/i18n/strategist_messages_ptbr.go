package i18n

// PTBRPhaseAnnouncements is the Portuguese (pt-BR) phase_announcements bundle.
var PTBRPhaseAnnouncements = PhaseAnnouncementsMessages{
	DiscoveryStarting:       "🎯 **Ranger{role_level_tag}:** Estou em campo. O reconhecimento começa agora.",
	DiscoveryDone:           "🎯 **Ranger{role_level_tag}:** Missão de campo concluída. Passando o dossiê para o Archivist.",
	RefinementStarting:      "📚 **Archivist{role_level_tag}:** Dossiê recebido. Iniciando análise sistemática.",
	RefinementDone:          "📚 **Archivist{role_level_tag}:** Refinamento concluído. O plano está pronto para avaliação.",
	ApprovalGateShown:       "🚦 **Gate:** O trabalho está feito. A decisão é sua — o que construímos merece materialização de documentação?",
	DocumentationStarting:   "🗡️ **Sniper{role_level_tag}:** Alvo confirmado. Silêncio — materializando documentação.",
	DocumentationTargetDone: "🗡️ **Sniper{role_level_tag}:** Alvo {done}/{total} concluído.",
	DocumentationDone:       "🗡️ **Sniper{role_level_tag}:** Concluído. Relatório entregue.",
	ScoutDone:               "🧭 **Scout{role_level_tag}:** Rota selecionada — {route}.",
}

// PTBRRuntime is the Portuguese (pt-BR) runtime message bundle.
var PTBRRuntime = RuntimeMessages{ //nolint:dupl
	IntakeSummary: "Missão recebida: {task_type} | delivery={delivery_strategy} |" +
		" compatibility={legacy_compatibility} | urgency={urgency} | intent={execution_intent}",
	IntakeIndexModeNone: "⚠️ **Nota de governança:** `intake_index_mode: none` — nenhum contexto de governança" +
		" foi indexado para esta query. Status atual do execution_gate: {execution_gate}.",

	RoleLevelHeader: "Fase: {phase_index}/04\n{role_name}\n{model_effort}",

	RoleStart: "{role_level_header}\n{role_emoji} **{role_title} [{mission_id}]:** {start_text}",
	RoleDone: "{role_level_header}\n{role_emoji} **{role_title} [{mission_id}]:** {done_text}\n" +
		"{artifact_label} {artifact_path}",
	RoleTaskDone: "{role_level_header}\n{role_emoji} **{role_title} [{mission_id}]:** {task_text}",
	RolePhrases: map[string]RolePhrase{
		"_default":  {Emoji: "🎭", StartText: "iniciando. skill={provider}", DoneText: "concluído.", ArtifactLabel: "Artefato em:"},
		"ranger":    {Emoji: "🎯", StartText: "iniciando reconhecimento. skill={provider}", DoneText: "missão de reconhecimento concluída.", ArtifactLabel: "Artefato em:"},
		"archivist": {Emoji: "📚", StartText: "iniciando análise e refinamento. skill={provider}", DoneText: "análise refinada.", ArtifactLabel: "Artefatos em:"},
		"sniper":    {Emoji: "🗡️", StartText: "alvo confirmado — iniciando materialização de documentação.", DoneText: "materialização de documentação concluída.", ArtifactLabel: "Relatório em:", TaskText: "alvo {done}/{total} materializado — {task_title}"},
	},

	ApprovalGatePrompt: "{role_level_header}\n🚦 **Gate [{mission_id}]:** AGUARDANDO CONFIRMAÇÃO\n\n" +
		"Plano em: {artifact_path}\n\n" +
		"Revisar e confirmar? (sim / nao / revisar)",

	OpportunityDetected: "⚔️ **Ataque de Oportunidade** — {count} item(s) detectado(s)\n{items_brief}",
	OpportunityGate:     "⚔️ **Side Quests disponíveis:**\n{manifest}\n\nAprovar? (sim / nao / selecionar)",
	OpportunitySignal:   "⚔️ **Ataque de oportunidade!** {count} item(s) detectado(s) — detalhes no gate.",

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
