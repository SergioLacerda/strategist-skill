package domain

// Transition groups classify sensitive mission-state changes.
const (
	TransitionGroupFinalizeAnalysis = "finalize_analysis"             // pending/refined -> done
	TransitionGroupReviewGate       = "review_gate"                   // analysis review before documentation
	TransitionGroupDocumentation    = "documentation_materialization" // sniper documentation writes
)

// MissionState represents the orchestrator FSM state.
type MissionState string

// Orchestrator finite-state machine states.
const (
	StateInit              MissionState = "INIT"
	StateOpportunityAttack MissionState = "OPPORTUNITY_ATTACK"
	StateOpportunityGate   MissionState = "OPPORTUNITY_GATE"
	StateOpportunityExec   MissionState = "OPPORTUNITY_EXEC"
	StateRefinement        MissionState = "REFINEMENT"
	StateApprovalGate      MissionState = "APPROVAL_GATE"
	StateExecution         MissionState = "EXECUTION"
	StateDoneAnalysis      MissionState = "DONE_ANALYSIS"
	StateDoneDelivery      MissionState = "DONE_DELIVERY"
	StateBlocked           MissionState = "BLOCKED"

	// Quick Draw route states (§5.0 pipeline).
	StateQuickDraw     MissionState = "QUICK_DRAW"
	StateQuickDrawGate MissionState = "QUICK_DRAW_GATE"
	StateQuickDrawDone MissionState = "QUICK_DRAW_DONE"

	// ADR stage states (§8 pipeline).
	StateADRGate1 MissionState = "ADR_GATE_1"
	StateADRGate2 MissionState = "ADR_GATE_2"
	StateADRDone  MissionState = "ADR_DONE"

	// Retry state for transient slot failures (protocol §Slot Failure Classification).
	StateRetrying MissionState = "RETRYING"

	// Critical Hit route states — fast path for low-risk doc/content tasks.
	StateDirectGate MissionState = "DIRECT_GATE"
	StateDirectExec MissionState = "DIRECT_EXEC"
	StateDirectDone MissionState = "DIRECT_DONE"
)

// TransitionEvent represents FSM/evaluator inputs.
type TransitionEvent string

// Transition events accepted by the orchestrator state machine.
const (
	EventManifestEmpty    TransitionEvent = "manifest_empty"
	EventManifestNonEmpty TransitionEvent = "manifest_non_empty"
	EventGateApproved     TransitionEvent = "gate_approved"
	EventGateDenied       TransitionEvent = "gate_denied"
	EventSniperDone       TransitionEvent = "sniper_done"
	EventArchivistNoTasks TransitionEvent = "archivist_done_no_tasks"
	EventArchivistTasks   TransitionEvent = "archivist_done_has_tasks"

	// Quick Draw route events (§3.1 detection → §5.0 execution).
	EventQuickDrawIntent  TransitionEvent = "quick_draw_intent"
	EventQuickDrawApprove TransitionEvent = "quick_draw_approved"
	EventQuickDrawDecline TransitionEvent = "quick_draw_declined"

	// ADR stage events (§8 pipeline).
	EventADRCriterionMet TransitionEvent = "adr_criterion_met"
	EventADRApproved     TransitionEvent = "adr_approved"
	EventADRDeclined     TransitionEvent = "adr_declined"

	// Slot failure classification events (protocol §Slot Failure Classification).
	EventSlotTransient TransitionEvent = "slot_transient_failure"
	EventSlotPermanent TransitionEvent = "slot_permanent_failure"

	// Sniper opportunity attack surfaced mid-documentation (§7 Opportunity Attack).
	EventSniperOA TransitionEvent = "sniper_opportunity_attack"

	// Critical Hit route events — fast path gate for direct_execute route.
	EventDirectHitIntent    TransitionEvent = "direct_hit_intent"
	EventDirectGateApproved TransitionEvent = "direct_gate_approved"
	EventDirectGateDeclined TransitionEvent = "direct_gate_declined"
)

// MissionPolicy controls whether guarded transitions are allowed.
// Sniper write scope is fixed by contract: documentation files only.
// Documentation materialization is always permitted after review gate acceptance.
type MissionPolicy struct{}

// TransitionDecision is the deterministic result of policy evaluation.
type TransitionDecision struct {
	Allowed bool
	Reason  string
	Status  string // allowed | policy_blocked | approval_required
	Policy  MissionPolicy
}

// DefaultMissionPolicy returns the fixed documentation policy for this skill.
func DefaultMissionPolicy() MissionPolicy {
	return MissionPolicy{}
}

// NormalizePolicy is a no-op kept for call-site compatibility.
func NormalizePolicy(p MissionPolicy) MissionPolicy {
	return p
}
