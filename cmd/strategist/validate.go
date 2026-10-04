package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

var validateRoot string

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the .strategist/ configuration tree",
	Long: `Validate all configuration files inside a .strategist/ directory.

Checks performed:
  - active.yaml: exists, valid YAML, required fields (mode, base_path, slots)
  - personas/*.yaml: each has tone_directive and phase_labels
  - roles/*.yaml: each has discovery, refinement, execution slots
  - knowledge.index.yaml: if present, valid YAML`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		root := validateRoot
		if root == "" {
			cwd, cwdErr := os.Getwd()
			if cwdErr != nil {
				return fmt.Errorf("validate: get cwd: %w", cwdErr)
			}
			discovered, _, discErr := findStrategistRoot(cwd)
			if discErr != nil {
				return fmt.Errorf("validate: runtime not found — run: strategist install")
			}
			root = discovered
		}
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		run := telemetry.MissionRunFromContext(ctx)
		if run != nil {
			run.MarkRanger()
		}
		ctx, span := telemetry.Tracer().Start(ctx, "strategist.validate",
			trace.WithAttributes(
				attribute.String(telemetry.AttrComponent, "validate"),
				attribute.String(telemetry.AttrRuntimeMode, "cli"),
				attribute.String(telemetry.AttrOutputProfile, "default"),
				attribute.String(telemetry.AttrTarget, root),
			),
		)
		defer span.End()

		report := application.ValidateWorkspace(root)
		errs := report.Errors
		checks := report.Checks

		if len(errs) > 0 {
			span.SetStatus(codes.Error, "validation failed")
			span.SetAttributes(attribute.StringSlice("strategist.validation.errors", errs))
			for _, e := range errs {
				fmt.Fprintf(os.Stderr, "  ✗ %s\n", e)
			}
			return fmt.Errorf("validate: %d error(s) in %s", len(errs), root)
		}

		if run != nil {
			run.AddLines(1)
		}
		slog.InfoContext(ctx, "[Strategist] validate complete",
			telemetry.AttrComponent, "validate",
			telemetry.AttrRuntimeMode, "cli",
			telemetry.AttrOutputProfile, "default",
			telemetry.AttrTarget, root,
			"checks", checks,
		)
		fmt.Printf("[Strategist] validate OK — %d check(s) passed (%s)\n", checks, root)
		return nil
	},
}

func init() {
	validateCmd.Flags().StringVar(&validateRoot, "root", "", "path to .strategist/ root (default: .strategist)")
}
