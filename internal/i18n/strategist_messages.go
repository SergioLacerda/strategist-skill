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
	RangerStart string
	RangerDone  string

	// Archivist (refinement) phase
	ArchivistStart string
	ArchivistDone  string

	// Sniper (documentation materialization) phase
	SniperStart    string
	SniperTaskDone string
	SniperDone     string

	// Approval gate
	ApprovalGatePrompt string

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

// ToMap converts the RuntimeMessages struct to a map[string]any with snake_case keys
// matching the content_by_lang.en conventions used in persona YAML files.
func (m RuntimeMessages) ToMap() map[string]any {
	return map[string]any{
		"intake_summary":            m.IntakeSummary,
		"intake_index_mode_none":    m.IntakeIndexModeNone,
		"ranger_start":              m.RangerStart,
		"ranger_done":               m.RangerDone,
		"archivist_start":           m.ArchivistStart,
		"archivist_done":            m.ArchivistDone,
		"sniper_start":              m.SniperStart,
		"sniper_task_done":          m.SniperTaskDone,
		"sniper_done":               m.SniperDone,
		"approval_gate_prompt":      m.ApprovalGatePrompt,
		"opportunity_detected":      m.OpportunityDetected,
		"opportunity_gate":          m.OpportunityGate,
		"opportunity_signal":        m.OpportunitySignal,
		"quick_draw_detected":       m.QuickDrawDetected,
		"quick_draw_gate":           m.QuickDrawGate,
		"quick_draw_success":        m.QuickDrawSuccess,
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
	}
}
