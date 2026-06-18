package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/SergioLacerda/strategist-skill/internal/stale"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

var checkStaleCmd = &cobra.Command{
	Use:   "check-stale <artifact.gz>",
	Short: "Check if a compiled artifact is stale (exit 0=fresh, exit 1=stale)",
	Args:  cobra.ExactArgs(1),
	RunE:  runCheckStale,
}

func runCheckStale(cmd *cobra.Command, args []string) error {
	artifactPath := args[0]

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	run := telemetry.MissionRunFromContext(ctx)
	if run != nil {
		run.MarkRanger()
		run.AddLines(1)
	}
	ctx, span := telemetry.Tracer().Start(ctx, "strategist.check_stale",
		trace.WithAttributes(
			attribute.String(telemetry.AttrComponent, "check_stale"),
			attribute.String(telemetry.AttrRuntimeMode, "cli"),
			attribute.String(telemetry.AttrOutputProfile, "default"),
			attribute.String(telemetry.AttrArtifact, artifactPath),
			attribute.String(telemetry.AttrArtifactPath, artifactPath),
		),
	)

	isStale, err := stale.Checker{}.IsStale(artifactPath)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.End()
		return fmt.Errorf("check-stale: %w", err)
	}

	span.SetAttributes(attribute.Bool(telemetry.AttrCacheHit, !isStale))
	slog.InfoContext(ctx, "[Strategist] check-stale result",
		telemetry.AttrComponent, "check_stale",
		telemetry.AttrRuntimeMode, "cli",
		telemetry.AttrOutputProfile, "default",
		telemetry.AttrArtifactPath, artifactPath,
		"stale", isStale,
	)
	span.End()

	if isStale {
		os.Exit(1)
	}
	return nil
}
