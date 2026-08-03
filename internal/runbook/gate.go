package runbook

import "errors"

// DecisionGate is a named condition that must hold for a runbook to
// proceed past a given point, per design.md § Sidecar Schema.
type DecisionGate struct {
	ID        string `yaml:"id"`
	Statement string `yaml:"statement"`
}

// ValidateDecisionGate checks a DecisionGate against its required fields.
func ValidateDecisionGate(g DecisionGate) error {
	var errs []error
	errs = append(errs, validateNamedValue("decision_gate_invalid", "id", g.ID, nil)...)
	errs = append(errs, validateNamedValue("decision_gate_invalid", "statement", g.Statement, nil)...)
	return errors.Join(errs...)
}
