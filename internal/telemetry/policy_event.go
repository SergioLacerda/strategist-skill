package telemetry

import (
	"fmt"
	"log/slog"
)

// PolicyEvent describes a canonical Strategist policy event payload.
type PolicyEvent struct {
	Phase           string
	Status          string
	Mission         string
	TransitionGroup string
	CorrelationID   string
	CanExecute      bool
	Reason          string
}

// FormatPolicyEvent returns a progress-contract compliant event line.
func FormatPolicyEvent(ev PolicyEvent) string {
	line := fmt.Sprintf("[Strategist] phase=%s status=%s mission=%s write_scope=workspace_and_docs can_execute=%t", ev.Phase, ev.Status, ev.Mission, ev.CanExecute)
	if ev.TransitionGroup != "" {
		line += " transition_group=" + ev.TransitionGroup
	}
	if ev.CorrelationID != "" {
		line += " correlation_id=" + ev.CorrelationID
	}
	if ev.Reason != "" {
		line += " reason=" + ev.Reason
	}
	return line
}

// EmitPolicyEvent logs a canonical policy event line through slog.
func EmitPolicyEvent(ev PolicyEvent) {
	attrs := []any{
		AttrPhase, ev.Phase,
		AttrStatus, ev.Status,
		AttrMissionID, ev.Mission,
		"strategist.write_scope", "workspace_and_docs",
		"strategist.can_execute", ev.CanExecute,
	}
	if ev.TransitionGroup != "" {
		attrs = append(attrs, AttrTransitionGroup, ev.TransitionGroup)
	}
	if ev.CorrelationID != "" {
		attrs = append(attrs, AttrCorrelationID, ev.CorrelationID)
	}
	if ev.Reason != "" {
		attrs = append(attrs, AttrReason, ev.Reason)
	}
	slog.Info(FormatPolicyEvent(ev), attrs...)
}
