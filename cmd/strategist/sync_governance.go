package main

import (
	"context"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

var (
	syncGovernanceRoot   string
	syncGovernanceSddDir string
	syncGovernanceDryRun bool
)

var syncGovernanceCmd = &cobra.Command{
	Use:   "sync-governance",
	Short: "Sync .strategist/skill.yaml with active SDD governance mandates",
	Long: `Reads .sdd/ governance mandates and reconciles .strategist/skill.yaml.

Checks performed:
  - Reads .sdd/metadata.json to verify governance fingerprint
  - Reads .sdd/source/governance-core.json to extract active mandates
  - Compares active mandates against compliance.mandates in skill.yaml
  - Applies missing governance fields (validation_policy, budget_policy, telemetry_policy)
  - Reports drift before applying changes

Use --dry-run to preview changes without writing.`,
	RunE: runSyncGovernanceCmd,
}

func runSyncGovernanceCmd(cmd *cobra.Command, _ []string) (retErr error) {
	applySyncGovernanceDefaults()

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	run := telemetry.MissionRunFromContext(ctx)
	if run != nil {
		run.MarkRanger()
	}
	ctx, span := telemetry.Tracer().Start(ctx, "strategist.sync_governance")
	defer func() {
		if retErr != nil {
			span.RecordError(retErr)
			span.SetStatus(codes.Error, retErr.Error())
		}
		span.End()
	}()

	report, err := runSyncGovernance(syncGovernanceRoot, syncGovernanceSddDir, syncGovernanceDryRun)
	if err != nil {
		return fmt.Errorf("sync-governance: %w", err)
	}

	span.SetAttributes(
		attribute.String(telemetry.AttrComponent, "sync_governance"),
		attribute.String(telemetry.AttrRuntimeMode, "cli"),
		attribute.String(telemetry.AttrOutputProfile, "default"),
		attribute.String(telemetry.AttrTarget, syncGovernanceRoot),
		attribute.Int(telemetry.AttrMandates, len(report.MandatesActive)),
		attribute.StringSlice("strategist.mandates.missing", report.MandatesMissing),
	)
	if run != nil {
		run.AddLines(1)
	}
	slog.InfoContext(ctx, "[Strategist] sync-governance complete",
		telemetry.AttrComponent, "sync_governance",
		telemetry.AttrRuntimeMode, "cli",
		telemetry.AttrOutputProfile, "default",
		telemetry.AttrTarget, syncGovernanceRoot,
		"fingerprint", report.GovernanceFingerprint,
		"active", len(report.MandatesActive),
		"missing", len(report.MandatesMissing),
	)

	printSyncReport(report, run)
	return nil
}

func applySyncGovernanceDefaults() {
	if syncGovernanceRoot == "" {
		syncGovernanceRoot = ".strategist"
	}
	if syncGovernanceSddDir == "" {
		syncGovernanceSddDir = ".sdd"
	}
}

func init() {
	syncGovernanceCmd.Flags().StringVar(&syncGovernanceRoot, "root", "", "path to .strategist/ root (default: .strategist)")
	syncGovernanceCmd.Flags().StringVar(&syncGovernanceSddDir, "sdd", "", "path to .sdd/ directory (default: .sdd)")
	syncGovernanceCmd.Flags().BoolVar(&syncGovernanceDryRun, "dry-run", false, "preview changes without writing")
}
