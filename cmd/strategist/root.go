// Package main is the entry point for the strategist CLI.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/integrity"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "strategist",
	Short: "Strategist skill CLI",
	Long:  "Strategist — install, compile, and manage the Strategist skill for Claude agents.",
}

func init() {
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		ctx := context.Background()
		run := telemetry.NewMissionRun(fmt.Sprintf("%s-%d", cmd.Name(), time.Now().UnixNano()))
		ctx = telemetry.WithMissionRun(ctx, run)
		run.MarkIntake()
		run.AddLines(1)
		slogLine := run.StartLine("local", ".strategist", ".strategist/active.yaml", "unknown", "local_default", "default")
		fmt.Println(slogLine)
		slog.InfoContext(ctx, slogLine,
			telemetry.AttrComponent, "root",
			telemetry.AttrRuntimeMode, "cli",
			telemetry.AttrOutputProfile, "default",
			telemetry.AttrMissionID, run.MissionID,
		)
		cmd.SetContext(ctx)
		if modified, err := integrity.IsModified(".strategist/active.yaml", ".strategist/.config.lock"); err == nil && modified {
			fmt.Fprintf(os.Stderr,
				"[Strategist] WARN: active.yaml was modified outside the CLI.\n"+
					"             Config integrity unverified. Re-run `strategist install` to acknowledge.\n")
		}
		return nil
	}
	rootCmd.PersistentPostRunE = func(cmd *cobra.Command, _ []string) error {
		telemetry.FinishMission(cmd.Context())
		return nil
	}
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(compileCmd)
	rootCmd.AddCommand(checkStaleCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(syncGovernanceCmd)
	rootCmd.AddCommand(versionCmd)
}

func execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(exitCodeFor(err))
	}
}

// exitCodeFor maps known error categories to distinct exit codes so that
// CI/CD pipelines can distinguish policy violations from generic failures.
//
//	0 — success
//	1 — generic / unknown error
//	2 — governance / policy violation
//	3 — stale artifact or config integrity error
func exitCodeFor(err error) int {
	switch {
	case errors.Is(err, domain.ErrPipelineBypassDetected):
		return 2
	case errors.Is(err, domain.ErrSourceStale), errors.Is(err, domain.ErrArtifactAbsent), errors.Is(err, domain.ErrManifestMissing):
		return 3
	default:
		return 1
	}
}
