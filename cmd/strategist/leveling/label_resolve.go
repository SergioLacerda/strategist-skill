package leveling

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	internal "github.com/SergioLacerda/strategist-skill/internal/leveling"
)

const (
	defaultLedgerName       = "role-levels.jsonl"
	defaultLedgerMaxRecords = 2000
	// gateStep is the Approval Gate. It is a pipeline step, not a role.
	gateStep = "gate"
)

// LabelRole resolves the level for opts.Role with the default role registry.
func LabelRole(load internal.PolicyLoader, cfg domain.LevelingConfig, ledgerPath string, opts LabelOptions) (LabelResult, error) {
	return LabelRoleWith(domain.DefaultRoleRegistry(), load, cfg, ledgerPath, opts)
}

// LabelRoleWith is LabelRole with an explicit role registry, which maps the
// role to the LEVELING policy role it uses.
func LabelRoleWith(reg domain.RoleRegistry, load internal.PolicyLoader, cfg domain.LevelingConfig, ledgerPath string, opts LabelOptions) (LabelResult, error) {
	return labelRoleWithMax(reg, load, cfg, ledgerPath, opts, defaultLedgerMaxRecords)
}

func labelRoleWithMax(reg domain.RoleRegistry, load internal.PolicyLoader, cfg domain.LevelingConfig, ledgerPath string, opts LabelOptions, maxRecords int) (LabelResult, error) {
	if result, reused, err := reuseRecordedLevel(ledgerPath, opts); err != nil {
		return LabelResult{}, err
	} else if reused {
		return result, nil
	}
	result := resolveLabelResult(reg, load, cfg, opts)
	if err := recordLabelResult(ledgerPath, opts, &result, maxRecords); err != nil {
		return LabelResult{}, err
	}
	return result, nil
}

func reuseRecordedLevel(ledgerPath string, opts LabelOptions) (LabelResult, bool, error) {
	if opts.Mission == "" || opts.Reason != "" {
		return LabelResult{}, false, nil
	}
	record, ok, err := internal.LatestRunRecord(ledgerPath, opts.Mission, opts.Role, opts.Run)
	if err != nil {
		return LabelResult{}, false, fmt.Errorf("leveling: read role level ledger: %w", err)
	}
	if !ok || record.Unknown() {
		return LabelResult{}, false, nil
	}
	return LabelResult{Level: record.Level, Reused: true}, true, nil
}

func resolveLabelResult(reg domain.RoleRegistry, load internal.PolicyLoader, cfg domain.LevelingConfig, opts LabelOptions) LabelResult {
	provider := opts.Provider
	if cfg.HostPassthrough() {
		provider = ""
	}
	host, rejected := internal.SanitizeHost(internal.Host{Model: opts.HostModel, Effort: opts.HostEffort})
	signals := internal.Signals{Ambiguity: opts.Ambiguity, Risk: opts.Risk, Scope: opts.Scope, Evidence: opts.Evidence}
	resolve := func() (internal.Level, error) {
		return internal.ResolveLevelLazy(load, provider, reg.PolicyRole(opts.Role), signals, host)
	}
	if provider == "" && !cfg.HostPassthrough() {
		// Automatic mode without --provider (the role on_start hook): infer the
		// provider from the host model so the policy can complete the level.
		resolve = func() (internal.Level, error) {
			return internal.ResolveLevelInferred(load, reg.PolicyRole(opts.Role), signals, host)
		}
	}
	level, err := resolve()
	if err != nil {
		return LabelResult{Level: internal.Level{Role: internal.NormalizeRole(opts.Role)}, Warning: joinWarnings(append(rejected, err.Error()))}
	}
	level.Role = internal.NormalizeRole(opts.Role)
	return LabelResult{Level: level, Warning: joinWarnings(rejected)}
}

func joinWarnings(warnings []string) string {
	return strings.Join(warnings, "; ")
}

func recordableLevel(ledgerPath string, opts LabelOptions, level internal.Level) (bool, error) {
	if internal.NormalizeRole(opts.Role) == gateStep {
		return false, nil
	}
	if !level.Unknown() || opts.Reason != "" {
		return true, nil
	}
	latest, found, err := internal.LatestRunRecord(ledgerPath, opts.Mission, opts.Role, opts.Run)
	if err != nil {
		return false, fmt.Errorf("leveling: read role level ledger: %w", err)
	}
	return !found || !latest.Unknown(), nil
}

func recordLabelResult(ledgerPath string, opts LabelOptions, result *LabelResult, maxRecords int) error {
	if opts.Mission == "" {
		return nil
	}
	if ok, err := recordableLevel(ledgerPath, opts, result.Level); err != nil || !ok {
		return err
	}
	result.Recorded, result.Reason = true, opts.Reason
	if err := internal.AppendRecord(ledgerPath, internal.Record{MissionID: opts.Mission, Run: opts.Run, Level: result.Level, Reason: opts.Reason}); err != nil {
		return fmt.Errorf("leveling: record role level: %w", err)
	}
	if _, err := internal.RotateLedgerIfLarge(ledgerPath, ledgerRotateBytes, maxRecords); err != nil {
		result.Warning = err.Error()
	}
	return nil
}

// ledgerRotateBytes is the ledger size past which `label` compacts it.
const ledgerRotateBytes = 512 * 1024

func ledgerName(deps Dependencies) string {
	if deps.LedgerName == "" {
		return defaultLedgerName
	}
	return deps.LedgerName
}

func rotateMax(deps Dependencies) int {
	if deps.RotateMax <= 0 {
		return defaultLedgerMaxRecords
	}
	return deps.RotateMax
}

func loadConfig(deps Dependencies, root string) (domain.LevelingConfig, string) {
	if deps.LoadConfig == nil {
		return domain.LevelingConfig{}, ""
	}
	return deps.LoadConfig(root)
}

func loadRegistry(deps Dependencies, root string) (domain.RoleRegistry, string) {
	if deps.LoadRegistry == nil {
		return domain.DefaultRoleRegistry(), ""
	}
	return deps.LoadRegistry(root)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
