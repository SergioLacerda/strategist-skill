package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/integrity"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

// humanStatusCommands are commands whose default output is a human-readable
// status display or a single machine-copyable value (e.g. `version`). They
// suppress the pipeline startup line and metrics.
var humanStatusCommands = map[string]bool{
	"check":          true,
	"treasure-chest": true,
	"metrics":        true,
	"handoff":        true,
	"provider":       true,
	"version":        true,
	"leveling":       true,
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

// warnIfConfigModified checks the sealed config lock against the actual
// .strategist runtime root (resolved the same way every other root command
// does, via findStrategistRoot), not a path hardcoded relative to cwd — so the
// warning also fires correctly when a command runs from a subdirectory.
func warnIfConfigModified() {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	strategistDir, _, err := findStrategistRoot(cwd)
	if err != nil {
		// No runtime root yet (e.g. before install) — nothing to warn about.
		return
	}

	activePath := filepath.Join(strategistDir, "active.yaml")
	lockPath := filepath.Join(strategistDir, ".config.lock")

	result, err := integrity.Check(activePath, lockPath)
	if err != nil {
		// Corrupt lock is surfaced but never blocks an ordinary command.
		fmt.Fprintf(os.Stderr,
			"[Strategist] WARN: active.yaml lock is corrupt (%v).\n"+
				"             Re-run `strategist install` to reseal it.\n", err)
		return
	}
	if !result.Modified {
		// Includes the "no lock yet" case (first install) — stay quiet.
		return
	}

	switch result.Reason {
	case integrity.ReasonUnmodified, integrity.ReasonLockMissing, integrity.ReasonLegacyLock:
		return
	case integrity.ReasonConfigMissing:
		fmt.Fprintf(os.Stderr,
			"[Strategist] WARN: active.yaml missing after lock.\n"+
				"             Re-run `strategist install` to restore it.\n")
	case integrity.ReasonPathMismatch:
		fmt.Fprintf(os.Stderr,
			"[Strategist] WARN: active.yaml lock path mismatch.\n"+
				"             Re-run `strategist install` to reseal the lock.\n")
	case integrity.ReasonMTimeMismatch, integrity.ReasonHashMismatch, integrity.ReasonSizeMismatch:
		fmt.Fprintf(os.Stderr,
			"[Strategist] WARN: active.yaml was modified outside the CLI (reason=%s).\n"+
				"             Config integrity unverified. Re-run `strategist install` to acknowledge.\n",
			result.Reason)
	}
}

// installRootHooks wires the pipeline startup banner, mission metrics and
// config-integrity warning around every command run.
func installRootHooks(root *cobra.Command) {
	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
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
	root.PersistentPostRunE = func(cmd *cobra.Command, _ []string) error {
		telemetry.FinishMission(cmd.Context())
		return nil
	}
}
