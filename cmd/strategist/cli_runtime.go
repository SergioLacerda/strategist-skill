package main

import (
	"context"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

func commandContext(cmd *cobra.Command) context.Context {
	if ctx := cmd.Context(); ctx != nil {
		return ctx
	}
	return context.Background()
}

func addMissionLines(ctx context.Context, lines int64) {
	if run := telemetry.MissionRunFromContext(ctx); run != nil {
		run.AddLines(lines)
	}
}
