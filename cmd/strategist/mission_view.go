package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/SergioLacerda/strategist-skill/internal/missionview"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
)

type missionViewOptions struct {
	Root, MissionID, Run string
	JSON                 bool
}

var missionViewCmd = newMissionViewCommand()

func newMissionViewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view",
		Short: "Render a read-only mission experience view",
		Long:  "Render lifecycle, roles, advisory confidence, Approval Gate outcome, and LEVELING provenance for one mission. This command never authorizes or advances a mission.",
		RunE:  func(cmd *cobra.Command, _ []string) error { return runMissionView(cmd, missionViewOptionsFrom(cmd)) },
	}
	cmd.Flags().String(flagRoot, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	cmd.Flags().String("mission-id", "", "mission identifier (required)")
	cmd.Flags().String("run", "", "select one explicit repeated role run")
	cmd.Flags().Bool("json", false, "emit strategist-mission-view/v1 JSON")
	return cmd
}

func missionViewOptionsFrom(cmd *cobra.Command) missionViewOptions {
	return missionViewOptions{Root: stringFlag(cmd, flagRoot, ""), MissionID: stringFlag(cmd, "mission-id", ""), Run: stringFlag(cmd, "run", ""), JSON: boolFlag(cmd, "json", false)}
}

func runMissionView(cmd *cobra.Command, opts missionViewOptions) error {
	if err := requireMissionID(opts.MissionID); err != nil {
		return err
	}
	root, _, err := cliutil.ResolveActiveBasePath(opts.Root)
	if err != nil {
		return fmt.Errorf("mission view: %w", err)
	}
	_, status, err := loadMission(root, opts.MissionID)
	if err != nil {
		return fmt.Errorf("mission view: %w", err)
	}
	view := loadMissionView(root, status, opts.Run)
	if opts.JSON {
		if err := json.NewEncoder(cmd.OutOrStdout()).Encode(view); err != nil {
			return fmt.Errorf("mission view: write output: %w", err)
		}
		return nil
	}
	if err := missionview.RenderHuman(cmd.OutOrStdout(), view); err != nil {
		return fmt.Errorf("mission view: write output: %w", err)
	}
	return nil
}

func loadMissionView(root string, status domain.MissionEngineStatus, run string) missionview.View {
	reg, regErr := domain.LoadRoleRegistry(filepath.Join(root, "roles"))
	if regErr != nil {
		reg = domain.DefaultRoleRegistry()
	}
	active, activeErr := cliutil.LoadActiveConfig(root)
	if activeErr != nil {
		active = domain.ActiveConfig{}
	}
	confidence, confidenceErr := telemetry.LoadConfidenceGateReviewForRun(root, status.MissionID, run)
	gateOutcome, gateErr := telemetry.GateOutcomeFor(root, status.MissionID)
	levels, levelsErr := leveling.ReadRecords(filepath.Join(root, "memory", roleLevelLedger))
	levels = filterLevelRecords(levels, status.MissionID)
	return missionview.Build(missionview.Input{Status: status, Registry: reg, SlotProviders: active.Slots, Confidence: confidence, ConfidenceError: confidenceErr, GateOutcome: gateOutcome, GateError: gateErr, Levels: levels, LevelsError: levelsErr, Run: run})
}
