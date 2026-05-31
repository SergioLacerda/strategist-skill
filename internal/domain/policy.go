package domain

// Legacy done scopes.
const (
	DoneScopeAnalise = "analise"
	DoneScopeEntrega = "entrega"
)

// Mission modes (single user-facing control).
const (
	MissionModeAnalise          = "analise"
	MissionModeEntregaRevisada  = "entrega_revisada"
	MissionModeEntregaExecutada = "entrega_executada"
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
	StateInit            MissionState = "INIT"
	StateHousekeeping    MissionState = "HOUSEKEEPING"
	StateOpportunityGate MissionState = "OPPORTUNITY_GATE"
	StateOpportunityExec MissionState = "OPPORTUNITY_EXEC"
	StateRefinement      MissionState = "REFINEMENT"
	StateApprovalGate    MissionState = "APPROVAL_GATE"
	StateExecution       MissionState = "EXECUTION"
	StateDoneAnalise     MissionState = "DONE_ANALISE"
	StateDoneEntrega     MissionState = "DONE_ENTREGA"
	StateBlocked         MissionState = "BLOCKED"
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
	if doneScope == DoneScopeAnalise {
		return MissionModeAnalise
	}
	if applyChanges {
		return MissionModeEntregaExecutada
	}
	return MissionModeEntregaRevisada
}

// NewMissionPolicy builds canonical policy from mission_mode.
func NewMissionPolicy(mode string) MissionPolicy {
	switch mode {
	case MissionModeAnalise:
		return MissionPolicy{Mode: mode, CanExecute: false, ExpectsDelivery: DoneScopeAnalise, DoneScope: DoneScopeAnalise, ApplyChanges: false}
	case MissionModeEntregaRevisada:
		return MissionPolicy{Mode: mode, CanExecute: false, ExpectsDelivery: DoneScopeEntrega, DoneScope: DoneScopeEntrega, ApplyChanges: false}
	case MissionModeEntregaExecutada:
		return MissionPolicy{Mode: mode, CanExecute: true, ExpectsDelivery: DoneScopeEntrega, DoneScope: DoneScopeEntrega, ApplyChanges: true}
	default:
		// Backward compatibility default preserves historical behavior.
		return NewMissionPolicy(MissionModeEntregaExecutada)
	}
}

// NormalizePolicy applies backward-compatible defaults/coherence.
func NormalizePolicy(p MissionPolicy) MissionPolicy {
	if p.Mode == "" {
		if p.DoneScope != "" || p.ApplyChanges {
			p.Mode = MissionModeFromLegacy(p.DoneScope, p.ApplyChanges)
		} else {
			p.Mode = MissionModeEntregaExecutada
		}
	}
	return NewMissionPolicy(p.Mode)
}
