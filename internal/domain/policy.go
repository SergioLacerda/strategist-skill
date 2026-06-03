package domain

// Legacy done scopes.
const (
	DoneScopeAnalysis = "analise"
	DoneScopeDelivery = "entrega"
)

// Mission modes (single user-facing control).
const (
	MissionModeAnalysis         = "analise"
	MissionModeRevisedDelivery  = "entrega_revisada"
	MissionModeExecutedDelivery = "entrega_executada"
)

// Transition groups classify sensitive mission-state changes.
const (
	TransitionGroupFinalizeAnalysis = "finalize_analysis" // pending/refined -> done
	TransitionGroupExecution        = "execution"         // sniper/code/git/config writes
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

	// Sniper opportunity attack surfaced mid-execution (§7 Opportunity Attack).
	EventSniperOA TransitionEvent = "sniper_opportunity_attack"
)

// MissionPolicy controls whether guarded transitions are allowed.
// MissionMode is canonical. DoneScope/ApplyChanges are derived for compatibility.
type MissionPolicy struct {
	Mode            string // analise | entrega_revisada | entrega_executada
	CanExecute      bool
	ExpectsDelivery string // analise | entrega
	DoneScope       string
	ApplyChanges    bool
}

// TransitionDecision is the deterministic result of policy evaluation.
type TransitionDecision struct {
	Allowed bool
	Reason  string
	Status  string // allowed | policy_blocked | approval_required
	Policy  MissionPolicy
}

// MissionModeFromLegacy maps the former 2-knob model to mission_mode.
func MissionModeFromLegacy(doneScope string, applyChanges bool) string {
	if doneScope == DoneScopeAnalysis {
		return MissionModeAnalysis
	}
	if applyChanges {
		return MissionModeExecutedDelivery
	}
	return MissionModeRevisedDelivery
}

// NewMissionPolicy builds canonical policy from mission_mode.
func NewMissionPolicy(mode string) MissionPolicy {
	switch mode {
	case MissionModeAnalysis:
		return MissionPolicy{Mode: mode, CanExecute: false, ExpectsDelivery: DoneScopeAnalysis, DoneScope: DoneScopeAnalysis, ApplyChanges: false}
	case MissionModeRevisedDelivery:
		return MissionPolicy{Mode: mode, CanExecute: false, ExpectsDelivery: DoneScopeDelivery, DoneScope: DoneScopeDelivery, ApplyChanges: false}
	case MissionModeExecutedDelivery:
		return MissionPolicy{Mode: mode, CanExecute: true, ExpectsDelivery: DoneScopeDelivery, DoneScope: DoneScopeDelivery, ApplyChanges: true}
	default:
		// Backward compatibility default preserves historical behavior.
		return NewMissionPolicy(MissionModeExecutedDelivery)
	}
}

// NormalizePolicy applies backward-compatible defaults/coherence.
func NormalizePolicy(p MissionPolicy) MissionPolicy {
	if p.Mode == "" {
		if p.DoneScope != "" || p.ApplyChanges {
			p.Mode = MissionModeFromLegacy(p.DoneScope, p.ApplyChanges)
		} else {
			p.Mode = MissionModeExecutedDelivery
		}
	}
	return NewMissionPolicy(p.Mode)
}
