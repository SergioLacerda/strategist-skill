package domain

// Transition groups classify sensitive mission-state changes.
const (
	TransitionGroupFinalizeAnalysis = "finalize_analysis"             // pending/refined -> done
	TransitionGroupReviewGate       = "review_gate"                   // analysis review before documentation
	TransitionGroupDocumentation    = "documentation_materialization" // sniper documentation writes
)

// Policy decision reasons and statuses are serialized into CLI output and telemetry.
const (
	PolicyReasonAllowed                = "allowed"
	PolicyReasonApprovalRequired       = "approval_required"
	PolicyReasonUnknownTransitionGroup = "unknown_transition_group"

	PolicyStatusAllowed          = "allowed"
	PolicyStatusApprovalRequired = "approval_required"
	PolicyStatusBlocked          = "policy_blocked"
)

// MissionState represents the orchestrator FSM state.
type MissionState string

// Orchestrator finite-state machine states.
const (
	StateInit             MissionState = "INIT"
	StateSideQuestScan    MissionState = "SIDE_QUEST_SCAN"
	StateSideQuestGate    MissionState = "SIDE_QUEST_GATE"
	StateSideQuestExec    MissionState = "SIDE_QUEST_EXEC"
	StateRefinement       MissionState = "REFINEMENT"
	StateApprovalGate     MissionState = "APPROVAL_GATE"
	StateHandoffChallenge MissionState = "HANDOFF_CHALLENGE"
	StateExecution        MissionState = "EXECUTION"
	StateDoneAnalysis     MissionState = "DONE_ANALYSIS"
	StateDoneDelivery     MissionState = "DONE_DELIVERY"
	StateBlocked          MissionState = "BLOCKED"

	// ADR stage states (§8 pipeline).
	StateADRGate1 MissionState = "ADR_GATE_1"
	StateADRGate2 MissionState = "ADR_GATE_2"
	StateADRDone  MissionState = "ADR_DONE"

	// Retry states for transient slot failures (protocol §Slot Failure Classification).
	// StateRetrying ("RETRYING", generic/unreachable legacy retry state — S8) was
	// removed: M016 check confirmed no code persists or parses the "RETRYING" string
	// (grep -rn '"RETRYING"' internal/ cmd/ matched only its own former declaration).
	StateRetryingRefinement MissionState = "RETRYING_REFINEMENT"
	StateRetryingExecution  MissionState = "RETRYING_EXECUTION"
	StateRetryingDirectExec MissionState = "RETRYING_DIRECT_EXEC"

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
	EventGateTimeout      TransitionEvent = "gate_timeout"
	// EventGateApprovedAnalysisOnly is the Approval Gate's acceptance of a
	// package with no documentation_target (every task is an
	// implementation_handoff): the analysis is delivered and nothing is left for
	// Sniper, so the mission ends instead of entering the handoff challenge.
	// Distinct from EventGateDenied, which records a rejection.
	EventGateApprovedAnalysisOnly TransitionEvent = "gate_approved_analysis_only"
	// EventHandoffNotApplicable repairs a mission that entered the handoff
	// challenge although its accepted package has no documentation_target
	// (for example, gate_approved was used before
	// EventGateApprovedAnalysisOnly existed): nothing can reach Sniper, so the
	// mission ends as analysis delivered.
	EventHandoffNotApplicable TransitionEvent = "handoff_challenge_not_applicable"
	// EventGateRevision is the Approval Gate's revision_requested outcome (D2):
	// contracts/narrative/05-approval-gate.md documents "Archivist revisits" as a
	// valid, non-error resolution, distinct from EventGateDenied (rejected/timeout,
	// terminal). See contracts/machine/mission-status.yaml's gate_revision_requested entry.
	EventGateRevision     TransitionEvent = "gate_revision_requested"
	EventHandoffPassed    TransitionEvent = "handoff_challenge_passed"
	EventHandoffFailed    TransitionEvent = "handoff_challenge_failed"
	EventHandoffExhausted TransitionEvent = "handoff_challenge_exhausted"
	EventSniperDone       TransitionEvent = "sniper_done"
	EventArchivistNoTasks TransitionEvent = "archivist_done_no_tasks"
	EventArchivistTasks   TransitionEvent = "archivist_done_has_tasks"

	// ADR stage events (§8 pipeline).
	EventADRCriterionMet TransitionEvent = "adr_criterion_met"
	EventADRApproved     TransitionEvent = "adr_approved"
	EventADRDeclined     TransitionEvent = "adr_declined"

	// Slot failure classification events (protocol §Slot Failure Classification).
	EventSlotTransient TransitionEvent = "slot_transient_failure"
	EventSlotPermanent TransitionEvent = "slot_permanent_failure"
	// EventRetryOK is the retry-success signal for the StateRetrying* states (S9):
	// previously EventManifestNonEmpty did double duty here and at the side-quest
	// manifest scan, two unrelated meanings on one token. Manifest events now
	// mean manifests only (Init, SideQuestScan).
	EventRetryOK TransitionEvent = "retry_ok"

	// Side quest surfaced during documentation materialization. This is the
	// generic side-quest gate, not Archivist's ADR-only Opportunity Attack routine.
	EventSniperSideQuest TransitionEvent = "sniper_side_quest_detected"

	// Critical Hit route events — fast path gate for direct_execute route.
	EventDirectHitIntent    TransitionEvent = "direct_hit_intent"
	EventDirectGateApproved TransitionEvent = "direct_gate_approved"
	EventDirectGateDeclined TransitionEvent = "direct_gate_declined"
)

// TransitionDecision is the deterministic result of policy evaluation.
type TransitionDecision struct {
	Allowed bool
	Reason  string
	Status  string // PolicyStatusAllowed | PolicyStatusBlocked | PolicyStatusApprovalRequired
}
