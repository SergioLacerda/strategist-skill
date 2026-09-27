package mission

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/SergioLacerda/strategist-skill/internal/missionview"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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
	tokenUsage, tokenUsageErr := readMissionTokenUsage(root, status.MissionID)
	handoffMetrics, handoffMetricsErr := readRefinementHandoffMetrics(root, status.MissionID)
	declaredBudget, budgetErr := readDeclaredTokenBudget(root)
	_ = budgetErr // absent/unparseable skill.yaml: display-only, not a diagnosed failure (no Diagnostic entry for it)
	return missionview.Build(missionview.Input{
		Status: status, Registry: reg, SlotProviders: active.Slots,
		Confidence: confidence, ConfidenceError: confidenceErr,
		GateOutcome: gateOutcome, GateError: gateErr,
		Levels: levels, LevelsError: levelsErr, Run: run,
		TokenUsage: tokenUsage, TokenUsageError: tokenUsageErr, DeclaredTokenBudget: declaredBudget,
		HandoffMetrics: handoffMetrics, HandoffMetricsError: handoffMetricsErr,
	})
}

func readRefinementHandoffMetrics(root, missionID string) ([]telemetry.RefinementHandoffLine, error) {
	all, err := telemetry.ReadRefinementHandoffLines(telemetry.HandoffMetricsPath(root))
	if err != nil {
		return nil, fmt.Errorf("read handoff metrics: %w", err)
	}
	filtered := make([]telemetry.RefinementHandoffLine, 0, len(all))
	for _, line := range all {
		if line.MissionID == missionID {
			filtered = append(filtered, line)
		}
	}
	return filtered, nil
}

// readMissionTokenUsage reads the mission-token-usage ledger and filters it
// to missionID — the ledger is shared across every mission that has ever
// called `mission report-usage` (F-T2, ADR-0057 § design.md task 3.3).
func readMissionTokenUsage(root, missionID string) ([]telemetry.MissionTokenUsageRecord, error) {
	all, err := telemetry.ReadMissionTokenUsage(telemetry.MissionTokenUsageHistoryPath(root))
	if err != nil {
		return nil, fmt.Errorf("read mission token usage: %w", err)
	}
	filtered := make([]telemetry.MissionTokenUsageRecord, 0, len(all))
	for _, rec := range all {
		if rec.MissionID == missionID {
			filtered = append(filtered, rec)
		}
	}
	return filtered, nil
}

// skillYAMLBudgetPolicy is the minimal shape this reads from skill.yaml —
// budget_policy.token_budget only, not the full document.
type skillYAMLBudgetPolicy struct {
	BudgetPolicy struct {
		TokenBudget string `yaml:"token_budget"`
	} `yaml:"budget_policy"`
}

// readDeclaredTokenBudget reads skill.yaml's declared budget_policy.token_budget
// verbatim — a qualitative tier (e.g. "high"), not a token count. A missing or
// unparseable skill.yaml returns "" rather than an error: this is
// display-only context for the token usage section, not a secondary
// authority whose absence merits a Diagnostic.
func readDeclaredTokenBudget(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "skill.yaml")) //nolint:gosec // G304: root is the discovered/validated .strategist runtime root
	if err != nil {
		return "", fmt.Errorf("read skill.yaml: %w", err)
	}
	var y skillYAMLBudgetPolicy
	if err := yaml.Unmarshal(data, &y); err != nil {
		return "", fmt.Errorf("parse skill.yaml: %w", err)
	}
	return y.BudgetPolicy.TokenBudget, nil
}
