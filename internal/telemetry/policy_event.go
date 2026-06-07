package telemetry

import (
	"fmt"
	"log/slog"
)

// PolicyEvent describes a canonical Strategist policy event payload.
type PolicyEvent struct {
	Phase              string
	Status             string
	Mission            string
	ExecutionMode      string
	GitPersistenceMode string
	CanExecute         bool
	Reason             string
}

// FormatPolicyEvent returns a progress-contract compliant event line.
func FormatPolicyEvent(ev PolicyEvent) string {
	line := fmt.Sprintf("[Strategist] phase=%s status=%s mission=%s execution_mode=%s git_persistence_mode=%s can_execute=%t", ev.Phase, ev.Status, ev.Mission, ev.ExecutionMode, ev.GitPersistenceMode, ev.CanExecute)
	if ev.Reason != "" {
		line += " reason=" + ev.Reason
	}
	return line
}

// EmitPolicyEvent logs a canonical policy event line through slog.
func EmitPolicyEvent(ev PolicyEvent) {
	slog.Info(FormatPolicyEvent(ev))
}
