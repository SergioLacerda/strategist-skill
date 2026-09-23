package leveling

import (
	"path/filepath"
	"strings"

	internal "github.com/SergioLacerda/strategist-skill/internal/leveling"
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
	reg, regWarning := loadRegistry(deps, root)
	cfg, cfgWarning := loadConfig(deps, root)
	result, err := labelRoleWithMax(reg, policyLoader(deps), cfg, filepath.Join(root, "memory", ledgerName(deps)), *opts, rotateMax(deps))
	if err != nil {
		return err
	}
	if result.Warning == "" {
		result.Warning = firstNonEmpty(cfgWarning, regWarning)
	}
	if result.Recorded && deps.EmitRoleLevel != nil {
		deps.EmitRoleLevel(cmd.Context(), opts.Mission, opts.Run, result.Level, result.Reason)
	}
	return WriteLabelResult(cmd, reg, result, *opts)
}

// ConfigureLabelFlags attaches all `leveling label` flags.
func ConfigureLabelFlags(cmd *cobra.Command, opts *LabelOptions) {
	f := cmd.Flags()
	f.StringVar(&opts.Role, "role", "ranger", "role: scout, ranger, archivist, sniper, gate, or transport")
	f.StringVar(&opts.Provider, "provider", "", "ranked provider used to complete a missing host value (optional)")
	f.StringVar(&opts.Mission, "mission", "", "mission id; records and reuses the level in .strategist/memory")
	f.StringVar(&opts.Run, "run", "", "run id for a repeated role in one mission (for example a revision loop); each run keeps its own level")
	f.StringVar(&opts.Reason, "reason", "", "record a new tuple with this reason (for example escalated)")
	f.StringVar(&opts.HostModel, "host-model", "", "model reported by the host; an unreplaced <placeholder> is ignored")
	f.StringVar(&opts.HostEffort, "host-effort", "", "effort reported by the host: "+strings.Join(internal.EffortTierNames(), ", "))
	f.StringVar(&opts.Ambiguity, "ambiguity", "", "ambiguity signal: low, medium, or high")
	f.StringVar(&opts.Risk, "risk", "", "risk signal: low, medium, or high")
	f.StringVar(&opts.Scope, "scope", "", "scope signal: bounded or cross_module")
	f.StringVar(&opts.Evidence, "evidence", "", "evidence signal: sufficient, insufficient, or conflicting")
	f.StringVar(&opts.Message, "message", "", "message to render after the label")
	f.IntVar(&opts.Width, "width", 0, "measured terminal width; 0 (unmeasurable) selects the stacked layout")
	f.BoolVar(&opts.JSON, "json", false, "emit JSON")
}

func policyLoader(deps Dependencies) internal.PolicyLoader {
	return func() (internal.Policy, error) {
		policy, _, err := deps.LoadPolicy()
		return policy, err
	}
}
