package mission

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/application"
	"github.com/SergioLacerda/strategist-skill/internal/cli/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/feats/opportunity_attack"
	"github.com/SergioLacerda/strategist-skill/internal/mission/gatecriteria"
	"github.com/SergioLacerda/strategist-skill/internal/mission/handoff"
	missionview "github.com/SergioLacerda/strategist-skill/internal/mission/view"
	"github.com/SergioLacerda/strategist-skill/internal/roles/registry"
	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/SergioLacerda/strategist-skill/internal/tools/leveling"
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
	if _, err := application.QueryMission(status); err != nil {
		return fmt.Errorf("mission view: %w", err)
	}
	view, err := LoadView(root, status, run, deps.Ledger, deps.FilterLevels)
	if err != nil {
		return fmt.Errorf("mission view: %w", err)
	}
	return renderView(cmd, view, asJSON)
}

// renderView writes the assembled view as JSON or human text, split out of
// RunView to keep its own cognitive complexity within budget.
func renderView(cmd *cobra.Command, view missionview.View, asJSON bool) error {
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
// to load are passed through as explicit errors, never as zero values. A nil
// filter is the same class of problem — a missing collaborator — so it fails
// closed with an explicit error too, rather than panicking or silently
// passing every record through unfiltered.
func LoadView(root string, status domain.MissionEngineStatus, run, ledger string, filter func([]leveling.Record, string) []leveling.Record) (missionview.View, error) {
	if filter == nil {
		return missionview.View{}, fmt.Errorf("FilterLevels dependency is nil")
	}
	reg := loadViewRoleRegistry(root)
	active := loadViewActiveConfig(root)
	confidence, confidenceErr := telemetry.LoadConfidenceGateReviewForRun(root, status.MissionID, run)
	gateOutcome, gateErr := telemetry.GateOutcomeFor(root, status.MissionID)
	levels, levelsErr := leveling.ReadRecords(filepath.Join(root, "memory", ledger))
	levels = filter(levels, status.MissionID)
	tokenUsage, tokenUsageErr := readMissionTokenUsage(root, status.MissionID)
	handoffMetrics, handoffMetricsErr := readRefinementHandoffMetrics(root, status.MissionID)
	sources := loadUsageSources(root, status.MissionID)
	declaredBudget, budgetErr := readDeclaredTokenBudget(root)
	_ = budgetErr // absent/unparseable skill.yaml: display-only, not a diagnosed failure (no Diagnostic entry for it)
	basePath := loadViewBasePath(root)
	criteria := gatecriteria.Build(gatecriteria.Input{Root: root, BasePath: basePath, MissionID: status.MissionID})
	candidates := loadViewOpportunities(root, basePath, status.MissionID)
	return missionview.Build(missionview.Input{
		Status: status, Registry: reg, SlotProviders: active.Slots,
		Confidence: confidence, ConfidenceError: confidenceErr,
		GateOutcome: gateOutcome, GateError: gateErr,
		Levels: levels, LevelsError: levelsErr, Run: run,
		TokenUsage: tokenUsage, TokenUsageError: tokenUsageErr, DeclaredTokenBudget: declaredBudget,
		HandoffMetrics: handoffMetrics, HandoffMetricsError: handoffMetricsErr,
		Lifecycle: sources.lifecycle, LifecycleError: sources.lifecycleErr,
		Usage: sources.usage, UsageError: sources.usageErr, KnownInvocations: sources.invocationIDs,
		Criteria:      &criteria,
		Opportunities: candidates,
	}), nil
}

func loadViewRoleRegistry(root string) domain.RoleRegistry {
	reg, err := roles.LoadRoleRegistry(filepath.Join(root, "roles"))
	if err != nil {
		return domain.DefaultRoleRegistry()
	}
	return reg
}

func loadViewActiveConfig(root string) domain.ActiveConfig {
	active, err := cliutil.LoadActiveConfig(root)
	if err != nil {
		return domain.ActiveConfig{}
	}
	return active
}

func loadViewBasePath(root string) string {
	_, basePath, err := cliutil.ResolveActiveBasePath(root)
	if err != nil {
		return ""
	}
	return basePath
}

func loadViewOpportunities(root, basePath, missionID string) []opportunity_attack.Candidate {
	manifest, err := opportunity_attack.LoadManifest(root, missionID)
	if err != nil {
		return nil
	}
	currentDigest, err := handoff.PackageDigest(filepath.Join(basePath, "refined", missionID))
	if err != nil || opportunity_attack.VerifyDigest(manifest, currentDigest) != nil {
		return nil
	}
	return manifest.Candidates
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
