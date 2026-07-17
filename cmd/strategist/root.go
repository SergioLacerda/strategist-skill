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

// humanStatusCommands are commands whose default output is a human-readable
// status display. They suppress the pipeline startup line and metrics.
var humanStatusCommands = map[string]bool{
	"initiative":     true,
	"check":          true,
	"treasure-chest": true,
}

var rootCmd = &cobra.Command{
	Use:   "strategist",
	Short: "Strategist skill CLI",
	Long:  "Strategist — install, compile, and manage the Strategist skill for Claude agents.",
}

// isHumanStatusCommand reports whether cmd or any of its ancestors is a
// human-status command. Subcommands (e.g. "add" under "treasure-chest")
// report their own Name(), not their parent's, so the check must walk up
// the command tree — otherwise only the bare parent command is silenced
// and every subcommand falls through to the verbose pipeline banner.
func isHumanStatusCommand(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if humanStatusCommands[c.Name()] {
			return true
		}
	}
	return false
}

func warnIfConfigModified() {
	if modified, err := integrity.IsModified(".strategist/active.yaml", ".strategist/.config.lock"); err == nil && modified {
		fmt.Fprintf(os.Stderr,
			"[Strategist] WARN: active.yaml was modified outside the CLI.\n"+
				"             Config integrity unverified. Re-run `strategist install` to acknowledge.\n")
	}
}

func init() {
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		ctx := context.Background()
		run := telemetry.NewMissionRun(fmt.Sprintf("%s-%d", cmd.Name(), time.Now().UnixNano()))
		ctx = telemetry.WithMissionRun(ctx, run)
		run.MarkIntake()
		run.AddLines(1)

		if isHumanStatusCommand(cmd) {
			run.SetSilent()
			cmd.SetContext(ctx)
			warnIfConfigModified()
			return nil
		}

		cwd, err := os.Getwd()
		if err != nil {
			cwd = "."
		}
		strategistDir, _, _ := findStrategistRoot(cwd) //nolint:errcheck
		if strategistDir == "" {
			strategistDir = ".strategist"
		}
		profile := resolveRuntimeProfile(strategistDir)
		slogLine := run.StartLine(profile.ProfileMode, profile.ProfilePath, profile.ActiveYAMLPath, profile.PersonaResolved, profile.Reason, profile.OutputProfile)
		// A single slog.InfoContext call is enough: Go's default handler already
		// prints this to stderr when telemetry is disabled, and the otelslog
		// handler forwards it to the OTel pipeline when enabled. An additional
		// fmt.Println here duplicated every line verbatim (see the "handshake"
		// bug where `treasure-chest add` printed this banner twice).
		slog.InfoContext(ctx, slogLine,
			telemetry.AttrComponent, "root",
			telemetry.AttrRuntimeMode, "cli",
			telemetry.AttrOutputProfile, profile.OutputProfile,
			telemetry.AttrMissionID, run.MissionID,
		)
		cmd.SetContext(ctx)
		warnIfConfigModified()
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

// requireStrategistDir returns an error if .strategist/active.yaml is absent in
// the current directory. Used by subcommands that depend on an installed workspace.
func requireStrategistDir() error {
	if _, err := os.Stat(".strategist/active.yaml"); os.IsNotExist(err) {
		return fmt.Errorf("[Strategist] error=not_installed\n→ Run: strategist install")
	}
	return nil
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
