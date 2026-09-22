package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
)

const roleLevelLedger = "role-levels.jsonl"

// ledgerRotateBytes is the ledger size past which `label` compacts it.
const ledgerRotateBytes = 512 * 1024

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
