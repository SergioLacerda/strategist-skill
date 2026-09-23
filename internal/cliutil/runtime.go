package cliutil

import (
	"context"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// CommandContext returns the command's context, or context.Background() when
// the command was invoked without one (for example directly from a test).
func CommandContext(cmd *cobra.Command) context.Context {
	if ctx := cmd.Context(); ctx != nil {
		return ctx
	}
	return context.Background()
}

// AddMissionLines adds lines to the MissionRun carried by ctx, if any.
func AddMissionLines(ctx context.Context, lines int64) {
	if run := telemetry.MissionRunFromContext(ctx); run != nil {
		run.AddLines(lines)
	}
}
