package missionview

import (
	"fmt"
	"io"
)

// RenderHuman writes a deterministic concise representation of a View.
func RenderHuman(w io.Writer, v View) error {
	if err := renderMission(w, v); err != nil {
		return err
	}
	if err := renderJourney(w, v.Journey); err != nil {
		return err
	}
	if err := renderConfidenceAndGate(w, v); err != nil {
		return err
	}
	if err := renderLevels(w, v.Leveling); err != nil {
		return err
	}
	return renderDiagnostics(w, v.Diagnostics)
}

func renderMission(w io.Writer, v View) error {
	return writef(w, "Mission\n  id: %s\n  phase: %s\n  state: %s\nJourney\n", v.MissionID, v.Lifecycle.Phase, v.Lifecycle.State)
}

func renderJourney(w io.Writer, journey []JourneyEntry) error {
	for _, step := range journey {
		provider := ""
		if step.Provider != "" {
			provider = fmt.Sprintf(" provider=%s", step.Provider)
		}
		if err := writef(w, "  %d: %s (%s)%s\n", step.Phase, step.ID, step.Kind, provider); err != nil {
			return err
		}
	}
	return nil
}

func renderConfidenceAndGate(w io.Writer, v View) error {
	if err := writef(w, "Confidence (advisory)\n  availability: %s\nApproval Gate\n  availability: %s\n", v.Confidence.Availability, v.ApprovalGate.Availability); err != nil {
		return err
	}
	if v.ApprovalGate.Outcome != "" {
		return writef(w, "  outcome: %s\n", v.ApprovalGate.Outcome)
	}
	return nil
}

func renderLevels(w io.Writer, levels LevelingSection) error {
	if err := writef(w, "LEVELING\n  availability: %s\n  selection: %s\n", levels.Availability, levels.Selection); err != nil {
		return err
	}
	for _, role := range levels.Roles {
		if err := renderLevelRole(w, role); err != nil {
			return err
		}
	}
	return nil
}

func renderLevelRole(w io.Writer, role LevelingRole) error {
	if role.Effective == nil {
		return writef(w, "  %s: unknown\n", role.Role)
	}
	level := role.Effective
	return writef(w, "  %s: %s source=%s model_source=%s effort_source=%s provider=%s capability=%s fallback_used=%t fallback_reason=%s policy_version=%d policy_digest=%s\n", level.Role, level.Label(), level.Source, level.ModelSource, level.EffortSource, level.Provider, level.Capability, level.FallbackUsed, level.FallbackReason, level.PolicyVersion, level.PolicyDigest)
}

func renderDiagnostics(w io.Writer, diagnostics []Diagnostic) error {
	if err := writef(w, "Diagnostics\n"); err != nil {
		return err
	}
	for _, diagnostic := range diagnostics {
		if err := writef(w, "  %s: %s\n", diagnostic.Reason, diagnostic.Action); err != nil {
			return err
		}
	}
	return nil
}

func writef(w io.Writer, format string, args ...any) error {
	if _, err := fmt.Fprintf(w, format, args...); err != nil {
		return fmt.Errorf("render mission view: %w", err)
	}
	return nil
}
