package i18n

// RuntimeMessages holds localized runtime strings emitted by Strategist agents during mission execution.
// These bundles are the canonical localization source for compiled content_by_lang
// runtime messages; protocol tokens remain centralized in reserved.go. Canonical
// persona YAML source remains English — pt-BR values live only here and are
// injected into content_by_lang.pt-BR at compile time (see internal/compile).
type RuntimeMessages struct {
	// Intake phase
	IntakeSummary       string
	IntakeIndexModeNone string

	// Ranger (discovery) phase

	// Archivist (refinement) phase

	// Sniper (documentation materialization) phase

	// Approval gate
	ApprovalGatePrompt string

	// Opportunity attack
	OpportunityDetected string
	OpportunityGate     string
	OpportunitySignal   string

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

	// RoleLevelHeader is the three-line header (phase counter, role, Model-Effort)
	// shown above role lines; omitted when the level is unknown. Fixed layout,
	// identical in every language. Resolve {model_effort} with
	// `strategist leveling label`.
	RoleLevelHeader string

	// Generic role message templates and per-role wording. The compile step
	// expands them once per registered role into the `<role>_start`,
	// `<role>_done` and `<role>_task_done` keys (see internal/compile), so a new
	// role needs no new template strings. Compile-time placeholders are
	// {role_title}, {role_emoji}, {start_text}, {done_text}, {artifact_label}
	// and {task_text}; the rest are runtime placeholders.
	// RolePhrases["_default"] words any role without its own.
	RoleStart    string
	RoleDone     string
	RoleTaskDone string
	RolePhrases  map[string]RolePhrase

	// Rendering helpers
	PhaseTimelineEntry string
	ArtifactEntry      string
}

// ToMap converts the RuntimeMessages struct to a map[string]any with snake_case keys
// matching the content_by_lang.en conventions used in persona YAML files.
func (m RuntimeMessages) ToMap() map[string]any {
	return map[string]any{
		"intake_summary":            m.IntakeSummary,
		"intake_index_mode_none":    m.IntakeIndexModeNone,
		"role_level_header":         m.RoleLevelHeader,
		"role_start":                m.RoleStart,
		"role_done":                 m.RoleDone,
		"role_task_done":            m.RoleTaskDone,
		"role_phrases":              rolePhrasesMap(m.RolePhrases),
		"approval_gate_prompt":      m.ApprovalGatePrompt,
		"opportunity_detected":      m.OpportunityDetected,
		"opportunity_gate":          m.OpportunityGate,
		"opportunity_signal":        m.OpportunitySignal,
		"treasure_chest_found":      m.TreasureChestFound,
		"side_quest_detected":       m.SideQuestDetected,
		"mission_checkpoint":        m.MissionCheckpoint,
		"compliance_summary":        m.ComplianceSummary,
		"execution_tasks_header":    m.ExecutionTasksHeader,
		"execution_task_line":       m.ExecutionTaskLine,
		"adr_opportunity":           m.AdrOpportunity,
		"adr_gate":                  m.AdrGate,
		"analysis_delivered_result": m.AnalysisDeliveredResult,
		"response_complete":         m.ResponseComplete,
		"mission_complete":          m.MissionComplete,
		"mission_metrics":           m.MissionMetrics,
		"phase_timeline_entry":      m.PhaseTimelineEntry,
		"artifact_entry":            m.ArtifactEntry,
	}
}

// PhaseAnnouncementsMessages holds localized per-phase mission narration lines
// (Ranger/Archivist/Gate/Sniper progress events), compiled into each persona's
// phase_announcements.<lang> field. This is a distinct compiled-artifact field
// from content_by_lang (RuntimeMessages above) — different keys, different
// wording, injected separately at compile time.
type PhaseAnnouncementsMessages struct {
	DiscoveryStarting       string
	DiscoveryDone           string
	RefinementStarting      string
	RefinementDone          string
	ApprovalGateShown       string
	DocumentationStarting   string
	DocumentationTargetDone string
	DocumentationDone       string
	ScoutDone               string
}

// ToMap converts PhaseAnnouncementsMessages to a map[string]any with snake_case
// keys matching the phase_announcements.en conventions used in persona YAML files.
func (m PhaseAnnouncementsMessages) ToMap() map[string]any {
	return map[string]any{
		"discovery_starting":        m.DiscoveryStarting,
		"discovery_done":            m.DiscoveryDone,
		"refinement_starting":       m.RefinementStarting,
		"refinement_done":           m.RefinementDone,
		"approval_gate_shown":       m.ApprovalGateShown,
		"documentation_starting":    m.DocumentationStarting,
		"documentation_target_done": m.DocumentationTargetDone,
		"documentation_done":        m.DocumentationDone,
		"scout_done":                m.ScoutDone,
	}
}

// RolePhrase is the wording of one role inside the generic role templates.
type RolePhrase struct {
	Emoji         string
	StartText     string
	DoneText      string
	ArtifactLabel string
	// TaskText is set only for a role that materializes tasks.
	TaskText string
}

func rolePhrasesMap(phrases map[string]RolePhrase) map[string]any {
	out := make(map[string]any, len(phrases))
	for role, phrase := range phrases {
		entry := map[string]any{
			"emoji": phrase.Emoji, "start_text": phrase.StartText,
			"done_text": phrase.DoneText, "artifact_label": phrase.ArtifactLabel,
		}
		if phrase.TaskText != "" {
			entry["task_text"] = phrase.TaskText
		}
		out[role] = entry
	}
	return out
}
