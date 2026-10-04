// Package policy holds the deterministic rules of the Integration Mechanism:
// per-attempt path selection and the circuit breaker. A provider answer is data;
// only these rules decide which path a call takes.
package policy

import "github.com/SergioLacerda/strategist-skill/internal/integration"

// Path is where a call is served.
type Path string

// Paths. Main is the existing flow, unchanged.
const (
	PathJEV  Path = "jev"
	PathMain Path = "main"
)

// Decision is the auditable result of one selection.
type Decision struct {
	Preferred Path
	Effective Path
	// Binding is the digest of the binding the decision was made for.
	Binding string
	// Reason is the state that kept the call off the preferred path; empty
	// when the preferred path is used.
	Reason integration.State
	// FellBack is true when the preferred path was wanted and main served it.
	FellBack bool
	// Suspect is true when the reason is a binding integrity failure. The call
	// still falls back to main, which runs every deterministic check, but the
	// consumer must surface the failure: a tampered binding never serves a call
	// and never goes unreported.
	Suspect bool
}

// Check evaluates one precondition and returns the state that fails it, or the
// empty state when it holds.
type Check func() integration.State

// Evaluate runs the checks in order and returns the first failing state. The
// order is the caller's priority: a disabled integration first, so a forged
// binding never matters when nothing would be called.
func Evaluate(checks ...Check) integration.State {
	for _, check := range checks {
		if state := check(); state != "" {
			return state
		}
	}
	return ""
}

// Select maps the state of the per-attempt checks to a path. JEV is always the
// preference and every failing state falls back to main, so the provider can
// never block a handoff. Disabled is a configured choice, not a fallback, and
// an integrity failure falls back flagged as suspect.
func Select(state integration.State, binding integration.Binding) Decision {
	decision := Decision{Preferred: PathJEV, Binding: binding.Digest(), Reason: state}
	switch state { //nolint:exhaustive // every non-empty state falls back to main
	case "":
		decision.Effective = PathJEV
	default:
		decision.Effective = PathMain
		decision.FellBack = state != integration.StateDisabled
		decision.Suspect = state == integration.StateBindingIntegrity
	}
	return decision
}
