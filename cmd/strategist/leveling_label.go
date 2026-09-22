package main

import (
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/spf13/cobra"
)

type levelingLabelOptions struct {
	Role, Provider, Mission, Run, Message, Reason string
	HostModel, HostEffort                         string
	Ambiguity, Risk, Scope, Evidence              string
	Width                                         int
	JSON                                          bool
}

// labelResult is the resolved level for one role plus how it was obtained.
type labelResult struct {
	Level  leveling.Level
	Reused bool
	// Recorded is true when a new tuple was appended to the role-level ledger.
	Recorded bool
	Warning  string
	// Reason is the escalation reason recorded with a new tuple, if any.
	Reason string
}

var levelingLabelOpts levelingLabelOptions

var levelingLabelCmd = newLevelingLabelCommandWithOptions(&levelingLabelOpts)

func newLevelingLabelCommand() *cobra.Command {
	return newLevelingLabelCommandWithOptions(&levelingLabelOptions{})
}

func newLevelingLabelCommandWithOptions(opts *levelingLabelOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "label",
		Short: "Resolve and record the model/effort label shown on a role's log lines",
		Long: `Resolve the level (model and effort) a role runs at and print the role-line
label. Precedence is the operator's manual choice (the "leveling:" block of
active.yaml), then host-reported --host-model/--host-effort, then the LEVELING
policy suggestion; the policy is read only when a value is still missing. With --mission the tuple is recorded in
.strategist/memory/role-levels.jsonl and reused by later calls for the same
mission and role, so every line of a phase agrees; pass --reason (for example
"escalated") to record a new tuple. The command never fails a mission: an
unknown level or a policy problem prints an unlabelled line and a warning.`,
		RunE: func(cmd *cobra.Command, _ []string) error { return runLevelingLabel(cmd, opts) },
	}
	configureLevelingLabelFlags(cmd, opts)
	return cmd
}

func runLevelingLabel(cmd *cobra.Command, opts *levelingLabelOptions) error {
	root, err := levelingWorkspaceRoot()
	if err != nil {
		return err
	}
	// Cheapest source first: the small `leveling:` block of active.yaml. The
	// policy is loaded lazily, only if a value is still missing.
	cfg, cfgWarning := readActiveLevelingConfig(root)
	reg, regWarning := loadRoleRegistry(root)
	result, err := labelRoleWith(reg, lazyLevelingPolicy, cfg, filepath.Join(root, "memory", roleLevelLedger), *opts)
	if err != nil {
		return err
	}
	if result.Warning == "" {
		result.Warning = firstNonEmpty(cfgWarning, regWarning)
	}
	if result.Recorded {
		emitRoleLevel(cmd.Context(), opts.Mission, opts.Run, result.Level, result.Reason)
	}
	return writeLabelResult(cmd, reg, result, *opts)
}

func configureLevelingLabelFlags(cmd *cobra.Command, opts *levelingLabelOptions) {
	f := cmd.Flags()
	f.StringVar(&opts.Role, "role", "ranger", "role: scout, ranger, archivist, sniper, gate, or transport")
	f.StringVar(&opts.Provider, "provider", "", "ranked provider used to complete a missing host value (optional)")
	f.StringVar(&opts.Mission, "mission", "", "mission id; records and reuses the level in .strategist/memory")
	f.StringVar(&opts.Run, "run", "", "run id for a repeated role in one mission (for example a revision loop); each run keeps its own level")
	f.StringVar(&opts.Reason, "reason", "", "record a new tuple with this reason (for example escalated)")
	f.StringVar(&opts.HostModel, "host-model", "", "model reported by the host")
	f.StringVar(&opts.HostEffort, "host-effort", "", "effort reported by the host")
	f.StringVar(&opts.Ambiguity, "ambiguity", "", "ambiguity signal: low, medium, or high")
	f.StringVar(&opts.Risk, "risk", "", "risk signal: low, medium, or high")
	f.StringVar(&opts.Scope, "scope", "", "scope signal: bounded or cross_module")
	f.StringVar(&opts.Evidence, "evidence", "", "evidence signal: sufficient, insufficient, or conflicting")
	f.StringVar(&opts.Message, "message", "", "message to render after the label")
	f.IntVar(&opts.Width, "width", 0, "measured terminal width; 0 (unmeasurable) selects the stacked layout")
	f.BoolVar(&opts.JSON, "json", false, "emit JSON")
}
