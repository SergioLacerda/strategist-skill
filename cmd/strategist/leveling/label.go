package leveling

import (
	"fmt"
	"path/filepath"
	"strings"

	levelingapp "github.com/SergioLacerda/strategist-skill/internal/application/leveling"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	internal "github.com/SergioLacerda/strategist-skill/internal/tools/leveling"
	"github.com/spf13/cobra"
)

// LabelOptions are the command flags for `strategist leveling label`.
type LabelOptions struct {
	Role, Provider, Mission, Run, Message, Reason string
	HostModel, HostEffort                         string
	Ambiguity, Risk, Scope, Evidence              string
	Width                                         int
	JSON                                          bool
}

// LabelResult is the resolved level for one role plus how it was obtained.
type LabelResult struct {
	Level  internal.Level
	Reused bool
	// Recorded is true when a new tuple was appended to the role-level ledger.
	Recorded bool
	Warning  string
	// Reason is the escalation reason recorded with a new tuple, if any.
	Reason string
	// Escalation is present when --reason requested a budget escalation.
	Escalation levelingapp.EscalationOutcome
}

// NewLabel creates the `leveling label` command.
func NewLabel(deps Dependencies, opts *LabelOptions) *cobra.Command {
	if opts == nil {
		opts = &LabelOptions{}
	}
	cmd := &cobra.Command{
		Use:   "label",
		Short: "Resolve and record the model/effort label shown on a role's log lines",
		RunE:  func(cmd *cobra.Command, _ []string) error { return RunLabel(cmd, deps, opts) },
	}
	ConfigureLabelFlags(cmd, opts)
	return cmd
}

// RunLabel resolves, records, emits, and renders the role level.
func RunLabel(cmd *cobra.Command, deps Dependencies, opts *LabelOptions) error {
	root, err := deps.WorkspaceRoot()
	if err != nil {
		return err
	}
	reg, cfg, result, err := resolveLabel(deps, root, opts)
	if err != nil {
		return err
	}
	if err := processLabelEscalation(cmd, deps, root, cfg, result); err != nil {
		return err
	}
	emitRoleLevel(cmd, deps, opts, result)
	return WriteLabelResult(cmd, reg, result, *opts)
}

func resolveLabel(deps Dependencies, root string, opts *LabelOptions) (domain.RoleRegistry, domain.LevelingConfig, LabelResult, error) {
	reg, regWarning := loadRegistry(deps, root)
	cfg, cfgWarning := loadConfig(deps, root)
	result, err := labelRoleWithMax(reg, policyLoader(deps), cfg, filepath.Join(root, "memory", ledgerName(deps)), *opts, rotateMax(deps))
	if err != nil {
		return reg, cfg, LabelResult{}, err
	}
	if result.Warning == "" {
		result.Warning = firstNonEmpty(cfgWarning, regWarning)
	}
	return reg, cfg, result, nil
}

func processLabelEscalation(cmd *cobra.Command, deps Dependencies, root string, cfg domain.LevelingConfig, result LabelResult) error {
	if !result.Escalation.Attempted {
		return nil
	}
	if err := recordBudgetEscalation(root, cfg, result); err != nil {
		return fmt.Errorf("leveling: record budget escalation: %w", err)
	}
	if deps.EmitBudgetEvent != nil && result.Escalation.Budget != nil {
		deps.EmitBudgetEvent(cmd.Context(), *result.Escalation.Budget, budgetEscalationEvent(result.Escalation), budgetEventReason(result))
	}
	return nil
}

func emitRoleLevel(cmd *cobra.Command, deps Dependencies, opts *LabelOptions, result LabelResult) {
	if result.Recorded && deps.EmitRoleLevel != nil {
		deps.EmitRoleLevel(cmd.Context(), opts.Mission, opts.Run, result.Level, result.Reason)
	}
}

func recordBudgetEscalation(root string, cfg domain.LevelingConfig, result LabelResult) error {
	budget := result.Escalation.Budget
	if budget == nil {
		budget = result.Level.Budget
	}
	if budget == nil {
		return nil
	}
	path, err := internal.ResolveBudgetLedgerPath(root, cfg)
	if err != nil {
		return fmt.Errorf("resolve budget ledger path: %w", err)
	}
	record := internal.NewBudgetEventRecord(*budget, budgetEscalationEvent(result.Escalation), budgetEventReason(result), "unverified", budget.MaxDelegations, 0)
	if err := internal.AppendBudgetEvent(path, record); err != nil {
		return fmt.Errorf("append budget escalation event: %w", err)
	}
	return nil
}

func budgetEventReason(result LabelResult) string {
	if result.Escalation.Granted {
		return result.Reason
	}
	return result.Escalation.Reason
}

func budgetEscalationEvent(outcome levelingapp.EscalationOutcome) internal.BudgetEventKind {
	if outcome.Granted {
		return internal.BudgetEventEscalation
	}
	if outcome.Reason == internal.ReasonLevelingBudgetEscalationExhausted {
		return internal.BudgetEventExhaustion
	}
	return internal.BudgetEventDenial
}

// ConfigureLabelFlags attaches all `leveling label` flags.
func ConfigureLabelFlags(cmd *cobra.Command, opts *LabelOptions) {
	f := cmd.Flags()
	f.StringVar(&opts.Role, "role", defaultString(opts.Role, "ranger"), "role: scout, ranger, archivist, sniper, gate, or transport")
	f.StringVar(&opts.Provider, "provider", opts.Provider, "ranked provider used to complete a missing host value (optional)")
	f.StringVar(&opts.Mission, "mission", opts.Mission, "mission id; records and reuses the level in .strategist/memory")
	f.StringVar(&opts.Run, "run", opts.Run, "run id for a repeated role in one mission (for example a revision loop); each run keeps its own level")
	f.StringVar(&opts.Reason, "reason", opts.Reason, "record a new tuple with this reason (for example escalated)")
	f.StringVar(&opts.HostModel, "host-model", opts.HostModel, "model reported by the host; an unreplaced <placeholder> is ignored")
	f.StringVar(&opts.HostEffort, "host-effort", opts.HostEffort, "effort reported by the host: "+strings.Join(internal.EffortTierNames(), ", "))
	f.StringVar(&opts.Ambiguity, "ambiguity", opts.Ambiguity, "ambiguity signal: low, medium, or high")
	f.StringVar(&opts.Risk, "risk", opts.Risk, "risk signal: low, medium, or high")
	f.StringVar(&opts.Scope, "scope", opts.Scope, "scope signal: bounded or cross_module")
	f.StringVar(&opts.Evidence, "evidence", opts.Evidence, "evidence signal: sufficient, insufficient, or conflicting")
	f.StringVar(&opts.Message, "message", opts.Message, "message to render after the label")
	f.IntVar(&opts.Width, "width", opts.Width, "measured terminal width; 0 (unmeasurable) selects the stacked layout")
	f.BoolVar(&opts.JSON, "json", opts.JSON, "emit JSON")
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func policyLoader(deps Dependencies) internal.PolicyLoader {
	return func() (internal.Policy, error) {
		policy, _, err := deps.LoadPolicy()
		return policy, err
	}
}
