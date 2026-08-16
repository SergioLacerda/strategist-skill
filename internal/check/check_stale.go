package check

import (
	"context"
	"encoding/json"
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

var (
	checkStaleJSON  bool
	checkStaleQuiet bool
)

var checkStaleCmd = &cobra.Command{
	Use:   "check-stale <artifact.gz>",
	Short: "Check if a compiled artifact is stale (exit 0=fresh, exit 1=stale)",
	Args:  cobra.ExactArgs(1),
	RunE:  runCheckStale,
}

func init() {
	checkStaleCmd.Flags().BoolVar(&checkStaleJSON, "json", false, "print stale-check result as JSON")
	checkStaleCmd.Flags().BoolVar(&checkStaleQuiet, "quiet", false, "suppress human-readable stale output")
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

	result, err := stale.Checker{}.Check(artifactPath)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.End()
		return fmt.Errorf("check-stale: %w", err)
	}

	span.SetAttributes(
		attribute.Bool(telemetry.AttrCacheHit, !result.Stale),
		attribute.String("strategist.stale.reason", string(result.Reason)),
	)
	slog.InfoContext(ctx, "[Strategist] check-stale result",
		telemetry.AttrComponent, "check_stale",
		telemetry.AttrRuntimeMode, "cli",
		telemetry.AttrOutputProfile, "default",
		telemetry.AttrArtifactPath, artifactPath,
		"stale", result.Stale,
		"reason", result.Reason,
	)
	span.End()

	if err := printCheckStaleResult(result); err != nil {
		return err
	}
	if result.Stale {
		os.Exit(1)
	}
	return nil
}

func printCheckStaleResult(result stale.Result) error {
	if checkStaleJSON {
		data, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("check-stale: marshal json: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}
	if result.Stale && !checkStaleQuiet {
		if result.SourcePath != "" {
			fmt.Printf("stale: %s path=%s\n", result.Reason, result.SourcePath)
			return nil
		}
		fmt.Printf("stale: %s artifact=%s\n", result.Reason, result.ArtifactPath)
	}
	return nil
}
