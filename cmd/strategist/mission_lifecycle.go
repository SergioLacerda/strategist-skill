package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
)

type missionLifecycleOptions struct {
	Root      string
	MissionID string
	Event     string
	JSON      bool
	Refs      []string
	Digests   []string
	MaxRefs   int
	MaxBytes  int
}

var missionStartCmd = newMissionStartCommand()

func newMissionStartCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "start", Short: "Start a mission", RunE: func(cmd *cobra.Command, _ []string) error {
		return runMissionStart(cmd, missionLifecycleOptionsFrom(cmd))
	}}
	configureMissionLifecycleFlags(cmd)
	return cmd
}

func runMissionStart(cmd *cobra.Command, opts missionLifecycleOptions) error {
	if err := requireMissionID(opts.MissionID); err != nil {
		return err
	}
	root, _, err := cliutil.ResolveActiveBasePath(opts.Root)
	if err != nil {
		return fmt.Errorf("mission start: %w", err)
	}
	if err := requireNoExistingMission(root, opts.MissionID); err != nil {
		return err
	}
	engine, status, err := domain.StartMission(domain.MissionStartRequest{MissionID: opts.MissionID})
	if err != nil {
		return fmt.Errorf("mission start: %w", err)
	}
	if err := saveMission(root, engine.Status()); err != nil {
		return fmt.Errorf("mission start: %w", err)
	}
	return writeMissionResult(cmd, opts.JSON, status)
}

func requireNoExistingMission(root, missionID string) error {
	if _, err := os.Stat(missionPath(root, missionID)); err == nil {
		return fmt.Errorf("mission start: mission %q already exists", missionID)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("mission start: inspect existing state: %w", err)
	}
	return nil
}

var missionStatusCmd = newMissionStatusCommand()

func newMissionStatusCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "status", Short: "Inspect a mission status", RunE: func(cmd *cobra.Command, _ []string) error {
		opts := missionLifecycleOptionsFrom(cmd)
		if err := requireMissionID(opts.MissionID); err != nil {
			return err
		}
		root, _, err := cliutil.ResolveActiveBasePath(opts.Root)
		if err != nil {
			return fmt.Errorf("mission status: %w", err)
		}
		_, status, err := loadMission(root, opts.MissionID)
		if err != nil {
			return fmt.Errorf("mission status: %w", err)
		}
		return writeMissionResult(cmd, opts.JSON, status)
	}}
	configureMissionLifecycleFlags(cmd)
	return cmd
}

var missionSubmitCmd = newMissionSubmitCommand()

func newMissionSubmitCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "submit", Short: "Submit one authoritative mission event", RunE: func(cmd *cobra.Command, _ []string) error {
		return runMissionSubmit(cmd, missionLifecycleOptionsFrom(cmd))
	}}
	configureMissionLifecycleFlags(cmd)
	cmd.Flags().String("event", "", "mission event to submit (required)")
	return cmd
}

func runMissionSubmit(cmd *cobra.Command, opts missionLifecycleOptions) error {
	if err := requireMissionID(opts.MissionID); err != nil {
		return err
	}
	if opts.Event == "" {
		return fmt.Errorf("mission submit: --event is required")
	}
	root, _, err := cliutil.ResolveActiveBasePath(opts.Root)
	if err != nil {
		return fmt.Errorf("mission submit: %w", err)
	}
	engine, _, err := loadMission(root, opts.MissionID)
	if err != nil {
		return fmt.Errorf("mission submit: %w", err)
	}
	status, err := engine.Submit(domain.MissionEngineEvent(opts.Event))
	if err != nil {
		return fmt.Errorf("mission submit: rejected: %w", err)
	}
	if err := saveMission(root, status); err != nil {
		return fmt.Errorf("mission submit: %w", err)
	}
	return writeMissionResult(cmd, opts.JSON, status)
}

var missionContextCmd = newMissionContextCommand()

func newMissionContextCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "context", Short: "Materialize declared mission context", RunE: func(cmd *cobra.Command, _ []string) error {
		return runMissionContext(cmd, missionLifecycleOptionsFrom(cmd))
	}}
	configureMissionLifecycleFlags(cmd)
	cmd.Flags().StringSlice("ref", nil, "declared workspace-relative context reference (repeatable)")
	cmd.Flags().StringSlice("digest", nil, "expected sha256 digest for each --ref, in the same order")
	cmd.Flags().Int("max-refs", domain.DefaultContextMaxReferences, "maximum number of references")
	cmd.Flags().Int("max-bytes", domain.DefaultContextMaxBytes, "maximum materialized bytes")
	return cmd
}

func runMissionContext(cmd *cobra.Command, opts missionLifecycleOptions) error {
	if err := requireMissionID(opts.MissionID); err != nil {
		return err
	}
	root, _, err := cliutil.ResolveActiveBasePath(opts.Root)
	if err != nil {
		return fmt.Errorf("mission context: %w", err)
	}
	if _, _, err := loadMission(root, opts.MissionID); err != nil {
		return fmt.Errorf("mission context: %w", err)
	}
	materialized, err := domain.MaterializeContext(filepath.Dir(root), contextReferencesFrom(opts), opts.MaxRefs, opts.MaxBytes)
	if err != nil {
		return fmt.Errorf("mission context: %w", err)
	}
	return writeMissionResult(cmd, opts.JSON, materialized)
}

func contextReferencesFrom(opts missionLifecycleOptions) []domain.ContextReference {
	refs := make([]domain.ContextReference, len(opts.Refs))
	for i, ref := range opts.Refs {
		refs[i] = domain.ContextReference{Ref: ref, Kind: "context"}
		if i < len(opts.Digests) {
			refs[i].Digest = opts.Digests[i]
		}
	}
	return refs
}

func configureMissionLifecycleFlags(cmd *cobra.Command) {
	cmd.Flags().String(flagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	cmd.Flags().String("mission-id", "", "mission identifier (required)")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON")
}

func missionLifecycleOptionsFrom(cmd *cobra.Command) missionLifecycleOptions {
	return missionLifecycleOptions{
		Root: stringFlag(cmd, flagRoot, ""), MissionID: stringFlag(cmd, "mission-id", ""),
		Event: stringFlag(cmd, "event", ""), JSON: boolFlag(cmd, "json", false),
		Refs: stringSliceFlag(cmd, "ref"), Digests: stringSliceFlag(cmd, "digest"),
		MaxRefs: intFlag(cmd, "max-refs", domain.DefaultContextMaxReferences), MaxBytes: intFlag(cmd, "max-bytes", domain.DefaultContextMaxBytes),
	}
}

func stringSliceFlag(cmd *cobra.Command, name string) []string {
	value, err := cmd.Flags().GetStringSlice(name)
	if err != nil {
		return nil
	}
	return value
}

func intFlag(cmd *cobra.Command, name string, fallback int) int {
	value, err := cmd.Flags().GetInt(name)
	if err != nil {
		return fallback
	}
	return value
}
