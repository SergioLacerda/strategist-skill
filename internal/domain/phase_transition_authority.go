package domain

import "fmt"

// PipelinePhase models the early-pipeline phases (bootstrap, intake,
// discovery, refinement) that state_machine.go's own header comment
// explicitly excludes from its stateTransitions table (S7) — until this
// type, those phases were enforced only by contract + progress events
// (prose), not by any transition table. See
// .analysis/pending/cli_refactor/20260915-cli-enforcement-refactor-refinement/design.md
// D3: this is the "full-pipeline FSM" follow-up state_machine.go's header
// comment used to point at
// .analysis/todo/analise-tecnica.md (a file that no longer exists in this
// tree as of this implementation — this type and its doc comments are now
// the authoritative restatement of that scope, superseding the stale
// pointer).
//
// PhaseTransitionAuthority is a separate, additive authority: it does not
// read, write, or otherwise reference stateTransitions, NextState, or
// RunStateMachine (state_machine.go), and it does not model
// gate/execution/retry/ADR/Critical-Hit — those remain exclusively
// state_machine.go's scope. The two authorities hand off at a single seam:
// PhaseRefinement here precedes StateRefinement there (Archivist's own work),
// which in turn transitions via EventArchivistTasks/EventArchivistNoTasks
// exactly as it already does today.
type PipelinePhase string

// Early-pipeline phases, in required order.
const (
	PhaseBootstrap    PipelinePhase = "BOOTSTRAP"
	PhaseIntake       PipelinePhase = "INTAKE"
	PhaseDiscovery    PipelinePhase = "DISCOVERY"
	PhaseRefinement   PipelinePhase = "REFINEMENT"
	PhaseApprovalGate PipelinePhase = "APPROVAL_GATE"
	PhaseExecution    PipelinePhase = "EXECUTION"
	PhaseDone         PipelinePhase = "DONE"
	PhaseBlocked      PipelinePhase = "BLOCKED"
)

// PhaseEvent is a phase-completion signal submitted to a
// PhaseTransitionAuthority.
type PhaseEvent string

// Phase-completion events accepted by PhaseTransitionAuthority.
const (
	EventBootstrapDone PhaseEvent = "bootstrap_done"
	EventIntakeDone    PhaseEvent = "intake_done"
	EventDiscoveryDone PhaseEvent = "discovery_done"
)

// phaseTransitions is intentionally its own map, never merged with
// stateTransitions (state_machine.go) — see the package-level doc comment
// above and D3's "extends, not replaces" decision.
var phaseTransitions = map[PipelinePhase]map[PhaseEvent]PipelinePhase{
	PhaseBootstrap: {EventBootstrapDone: PhaseIntake},
	PhaseIntake:    {EventIntakeDone: PhaseDiscovery},
	PhaseDiscovery: {EventDiscoveryDone: PhaseRefinement},
	// PhaseRefinement has no outgoing transition here: completion hands off
	// to state_machine.go's StateRefinement, which this authority does not
	// model or duplicate.
}

// ErrOutOfOrderPhaseSubmit is returned when a PhaseEvent is submitted from a
// PipelinePhase it is not declared for — e.g. submitting discovery-done while
// still in PhaseBootstrap (skipping intake).
type ErrOutOfOrderPhaseSubmit struct {
	Current PipelinePhase
	Event   PhaseEvent
}

func (e ErrOutOfOrderPhaseSubmit) Error() string {
	return fmt.Sprintf("phase transition authority: event %q is not valid from phase %q", e.Event, e.Current)
}

// PhaseTransitionAuthority enforces bootstrap→intake→discovery→refinement
// ordering by rejecting out-of-order or unrecognized submits, rather than
// self-looping on them. This is deliberately different from
// state_machine.go's NextState, which self-loops on an unhandled event: gate/
// execution mechanics tolerate a self-loop as a no-op, while an out-of-order
// phase submit is exactly the drift this authority exists to catch.
type PhaseTransitionAuthority struct {
	current PipelinePhase
}

// NewPhaseTransitionAuthority starts a new authority at PhaseBootstrap.
func NewPhaseTransitionAuthority() *PhaseTransitionAuthority {
	return &PhaseTransitionAuthority{current: PhaseBootstrap}
}

// Current returns the authority's current phase.
func (a *PhaseTransitionAuthority) Current() PipelinePhase {
	return a.current
}

// Submit applies event from the authority's current phase. On success it
// advances Current() and returns the new phase. On an out-of-order or
// unrecognized event it returns ErrOutOfOrderPhaseSubmit and leaves Current()
// unchanged.
func (a *PhaseTransitionAuthority) Submit(event PhaseEvent) (PipelinePhase, error) {
	transitions, ok := phaseTransitions[a.current]
	if !ok {
		return a.current, ErrOutOfOrderPhaseSubmit{Current: a.current, Event: event}
	}
	next, ok := transitions[event]
	if !ok {
		return a.current, ErrOutOfOrderPhaseSubmit{Current: a.current, Event: event}
	}
	a.current = next
	return a.current, nil
}
