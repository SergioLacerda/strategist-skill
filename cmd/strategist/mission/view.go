package mission

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

// ViewDependencies injects root resolution, mission loading and LEVELING
// ledger filtering for `mission view`.
type ViewDependencies struct {
	RootFlag, Ledger string
	RequireMissionID func(string) error
	ResolveBasePath  func(string) (string, string, error)
	Load             func(string, string) (*domain.MissionEngine, domain.MissionEngineStatus, error)
	FilterLevels     func([]leveling.Record, string) []leveling.Record
}

// NewView builds `mission view`.
func NewView(deps ViewDependencies) *cobra.Command {
	var root, missionID, run string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "view",
		Short: "Render a read-only mission experience view",
		Long:  "Render lifecycle, roles, advisory confidence, Approval Gate outcome, and LEVELING provenance for one mission. This command never authorizes or advances a mission.",
	}
	f := cmd.Flags()
	f.StringVar(&root, deps.RootFlag, "", "path to .strategist/ root (default: auto-discovered from CWD)")
	f.StringVar(&missionID, "mission-id", "", "mission identifier (required)")
	f.StringVar(&run, "run", "", "select one explicit repeated role run")
	f.BoolVar(&asJSON, "json", false, "emit strategist-mission-view/v1 JSON")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error { return RunView(cmd, deps, root, missionID, run, asJSON) }
	return cmd
}

// RunView renders the read-only mission view as human text or JSON.
func RunView(cmd *cobra.Command, deps ViewDependencies, rootInput, missionID, run string, asJSON bool) error {
	if err := deps.RequireMissionID(missionID); err != nil {
		return err
	}
	root, _, err := deps.ResolveBasePath(rootInput)
	if err != nil {
		return fmt.Errorf("mission view: %w", err)
	}
	_, status, err := deps.Load(root, missionID)
	if err != nil {
		return fmt.Errorf("mission view: %w", err)
	}
	view := LoadView(root, status, run, deps.Ledger, deps.FilterLevels)
	if asJSON {
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

// LoadView assembles the missionview.View input; secondary sources that fail
// to load are passed through as explicit errors, never as zero values.
func LoadView(root string, status domain.MissionEngineStatus, run, ledger string, filter func([]leveling.Record, string) []leveling.Record) missionview.View {
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
	levels, levelsErr := leveling.ReadRecords(filepath.Join(root, "memory", ledger))
	levels = filter(levels, status.MissionID)
	return missionview.Build(missionview.Input{Status: status, Registry: reg, SlotProviders: active.Slots, Confidence: confidence, ConfidenceError: confidenceErr, GateOutcome: gateOutcome, GateError: gateErr, Levels: levels, LevelsError: levelsErr, Run: run})
}
