package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
	"github.com/spf13/cobra"
)

const roleLevelLedger = "role-levels.jsonl"

// ledgerRotateBytes is the ledger size past which `label` compacts it.
const ledgerRotateBytes = 512 * 1024

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

var levelingLabelCmd = &cobra.Command{
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
	RunE: func(cmd *cobra.Command, _ []string) error {
		root, err := levelingWorkspaceRoot()
		if err != nil {
			return err
		}
		// Cheapest source first: the small `leveling:` block of active.yaml. The
		// policy is loaded lazily, only if a value is still missing.
		cfg, cfgWarning := readActiveLevelingConfig(root)
		reg, regWarning := loadRoleRegistry(root)
		result, err := labelRoleWith(reg, lazyLevelingPolicy, cfg, filepath.Join(root, "memory", roleLevelLedger), levelingLabelOpts)
		if err != nil {
			return err
		}
		if result.Warning == "" {
			result.Warning = firstNonEmpty(cfgWarning, regWarning)
		}
		if result.Recorded {
			emitRoleLevel(cmd.Context(), levelingLabelOpts.Mission, levelingLabelOpts.Run, result.Level, result.Reason)
		}
		return writeLabelResult(cmd, reg, result, levelingLabelOpts)
	},
}

func lazyLevelingPolicy() (leveling.Policy, error) {
	policy, _, err := loadLevelingPolicy()
	return policy, err
}

// readActiveLevelingConfig reads the `leveling:` block through the shared
// active.yaml loader. A missing or unreadable file, or an invalid block,
// degrades to automatic with a warning: labelling never blocks a mission.
func readActiveLevelingConfig(root string) (domain.LevelingConfig, string) {
	active, err := cliutil.LoadActiveConfig(root)
	if errors.Is(err, os.ErrNotExist) {
		return domain.LevelingConfig{}, ""
	}
	if err != nil {
		return domain.LevelingConfig{}, err.Error()
	}
	if err := active.Leveling.Validate(); err != nil {
		return domain.LevelingConfig{}, err.Error()
	}
	return active.Leveling, ""
}

// labelRole resolves the level for opts.Role, reusing the mission's recorded
// level unless opts.Reason asks for a new tuple. The policy is loaded through
// load only when manual, host and reported values leave a gap. Policy problems
// degrade to an unknown level with a warning instead of an error.
func labelRole(load leveling.PolicyLoader, cfg domain.LevelingConfig, ledgerPath string, opts levelingLabelOptions) (labelResult, error) {
	return labelRoleWith(domain.DefaultRoleRegistry(), load, cfg, ledgerPath, opts)
}

// labelRoleWith is labelRole with an explicit role registry, which maps the role
// to the LEVELING policy role it uses (its `leveling` key, default the role id).
func labelRoleWith(reg domain.RoleRegistry, load leveling.PolicyLoader, cfg domain.LevelingConfig, ledgerPath string, opts levelingLabelOptions) (labelResult, error) {
	if result, reused, err := reuseRecordedLevel(ledgerPath, opts); err != nil {
		return labelResult{}, err
	} else if reused {
		return result, nil
	}
	result := resolveLabelResult(reg, load, cfg, opts)
	if err := recordLabelResult(ledgerPath, opts, &result); err != nil {
		return labelResult{}, err
	}
	return result, nil
}

func reuseRecordedLevel(ledgerPath string, opts levelingLabelOptions) (labelResult, bool, error) {
	if opts.Mission == "" || opts.Reason != "" {
		return labelResult{}, false, nil
	}
	record, ok, err := leveling.LatestRunRecord(ledgerPath, opts.Mission, opts.Role, opts.Run)
	if err != nil {
		return labelResult{}, false, fmt.Errorf("leveling: read role level ledger: %w", err)
	}
	if !ok || record.Unknown() {
		return labelResult{}, false, nil
	}
	return labelResult{Level: record.Level, Reused: true}, true, nil
}

func resolveLabelResult(reg domain.RoleRegistry, load leveling.PolicyLoader, cfg domain.LevelingConfig, opts levelingLabelOptions) labelResult {
	manual, _ := cfg.Choice(opts.Role)
	level, err := leveling.ResolveLevelLazy(load, opts.Provider, reg.PolicyRole(opts.Role), leveling.Signals{
		Ambiguity: opts.Ambiguity, Risk: opts.Risk, Scope: opts.Scope, Evidence: opts.Evidence,
	}, leveling.Manual{Model: manual.Model, Effort: manual.Effort}, leveling.Host{Model: opts.HostModel, Effort: opts.HostEffort})
	if err != nil {
		return labelResult{Level: leveling.Level{Role: opts.Role}, Warning: err.Error()}
	}
	level.Role = opts.Role // the policy role only selects the criteria; the label names the real role
	return labelResult{Level: level}
}

func recordLabelResult(ledgerPath string, opts levelingLabelOptions, result *labelResult) error {
	if opts.Mission == "" {
		return nil
	}
	result.Recorded, result.Reason = true, opts.Reason
	if err := leveling.AppendRecord(ledgerPath, leveling.Record{MissionID: opts.Mission, Run: opts.Run, Level: result.Level, Reason: opts.Reason}); err != nil {
		return fmt.Errorf("leveling: record role level: %w", err)
	}
	// Keep the ledger bounded. A rotation problem never fails labelling: the
	// tuple was already recorded.
	if _, err := leveling.RotateLedgerIfLarge(ledgerPath, ledgerRotateBytes, defaultLedgerMaxRecords); err != nil {
		result.Warning = err.Error()
	}
	return nil
}

func init() {
	f := levelingLabelCmd.Flags()
	f.StringVar(&levelingLabelOpts.Role, "role", "ranger", "role: scout, ranger, archivist, sniper, gate, or transport")
	f.StringVar(&levelingLabelOpts.Provider, "provider", "", "ranked provider used to complete a missing host value (optional)")
	f.StringVar(&levelingLabelOpts.Mission, "mission", "", "mission id; records and reuses the level in .strategist/memory")
	f.StringVar(&levelingLabelOpts.Run, "run", "", "run id for a repeated role in one mission (for example a revision loop); each run keeps its own level")
	f.StringVar(&levelingLabelOpts.Reason, "reason", "", "record a new tuple with this reason (for example escalated)")
	f.StringVar(&levelingLabelOpts.HostModel, "host-model", "", "model reported by the host")
	f.StringVar(&levelingLabelOpts.HostEffort, "host-effort", "", "effort reported by the host")
	f.StringVar(&levelingLabelOpts.Ambiguity, "ambiguity", "", "ambiguity signal: low, medium, or high")
	f.StringVar(&levelingLabelOpts.Risk, "risk", "", "risk signal: low, medium, or high")
	f.StringVar(&levelingLabelOpts.Scope, "scope", "", "scope signal: bounded or cross_module")
	f.StringVar(&levelingLabelOpts.Evidence, "evidence", "", "evidence signal: sufficient, insufficient, or conflicting")
	f.StringVar(&levelingLabelOpts.Message, "message", "", "message to render after the label")
	f.IntVar(&levelingLabelOpts.Width, "width", 0, "measured terminal width; 0 (unmeasurable) selects the stacked layout")
	f.BoolVar(&levelingLabelOpts.JSON, "json", false, "emit JSON")
	levelingCmd.AddCommand(levelingLabelCmd)
}

// loadRoleRegistry loads the workspace's roles/*.yaml over the built-in registry.
// A broken role file degrades to the built-ins with a warning: labelling never
// blocks a mission.
func loadRoleRegistry(root string) (domain.RoleRegistry, string) {
	reg, err := domain.LoadRoleRegistry(filepath.Join(root, "roles"))
	if err != nil {
		return domain.DefaultRoleRegistry(), err.Error()
	}
	return reg, ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
