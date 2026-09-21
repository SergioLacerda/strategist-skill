package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// emitRoleLevel emits the structured role_level_resolved event (DEBUG in
// output-profiles/emit-taxonomy.yaml) carrying role, model, effort and level_source
// for telemetry. It uses the caller's inherited context; callers must provide
// a non-nil context rather than creating a new background context here.
// Unknown values are emitted empty, the null of a log attribute.
func emitRoleLevel(ctx context.Context, missionID, run string, level leveling.Level, reason string) {
	attrs := []any{
		telemetry.AttrComponent, "leveling",
		telemetry.AttrMissionID, missionID,
		telemetry.AttrRole, level.Role,
		telemetry.AttrModel, level.Model,
		telemetry.AttrEffort, level.Effort,
		telemetry.AttrLevelSource, level.Source,
	}
	if run != "" {
		attrs = append(attrs, telemetry.AttrRoleRun, run)
	}
	if reason != "" {
		attrs = append(attrs, telemetry.AttrReason, reason)
	}
	slog.DebugContext(ctx, "[Strategist] role_level_resolved", attrs...)
}

func writeLabelResult(cmd *cobra.Command, reg domain.RoleRegistry, result labelResult, opts levelingLabelOptions) error {
	if err := writeLabelWarning(cmd, result.Warning); err != nil {
		return err
	}
	rendered := leveling.RenderWith(reg, result.Level, opts.Message, opts.Width)
	if opts.JSON {
		return writeJSONLabel(cmd, result, rendered)
	}
	return writePlainLabel(cmd, result.Level.Label(), rendered, opts.Message != "")
}

func writeLabelWarning(cmd *cobra.Command, warning string) error {
	if warning == "" {
		return nil
	}
	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "warning: level unavailable: %s\n", warning); err != nil {
		return fmt.Errorf("leveling: write label: %w", err)
	}
	return nil
}

func writeJSONLabel(cmd *cobra.Command, result labelResult, rendered string) error {
	payload := struct {
		leveling.Level
		Label    string `json:"label"`
		Tag      string `json:"tag"`
		Rendered string `json:"rendered"`
		Reused   bool   `json:"reused"`
	}{result.Level, result.Level.Label(), result.Level.Tag(), rendered, result.Reused}
	if err := json.NewEncoder(cmd.OutOrStdout()).Encode(payload); err != nil {
		return fmt.Errorf("leveling: write label: %w", err)
	}
	return nil
}

func writePlainLabel(cmd *cobra.Command, label, rendered string, useRendered bool) error {
	if useRendered {
		label = rendered
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), label); err != nil {
		return fmt.Errorf("leveling: write label: %w", err)
	}
	return nil
}
