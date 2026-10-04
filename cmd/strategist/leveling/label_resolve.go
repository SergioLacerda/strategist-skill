package leveling

import (
	"fmt"

	levelingapp "github.com/SergioLacerda/strategist-skill/internal/application/leveling"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	internal "github.com/SergioLacerda/strategist-skill/internal/tools/leveling"
)

const (
	defaultLedgerName       = "role-levels.jsonl"
	defaultLedgerMaxRecords = 2000
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
	result, err := levelingapp.ResolveLevel(reg, load, cfg, ledgerPath, levelingapp.LevelingInput{
		Role: opts.Role, Provider: opts.Provider, Mission: opts.Mission, Run: opts.Run,
		HostModel: opts.HostModel, HostEffort: opts.HostEffort, Ambiguity: opts.Ambiguity,
		Risk: opts.Risk, Scope: opts.Scope, Evidence: opts.Evidence, Reason: opts.Reason,
	}, maxRecords)
	if err != nil {
		return LabelResult{}, fmt.Errorf("resolve level: %w", err)
	}
	return LabelResult{Level: result.Level, Reused: result.Reused, Recorded: result.Recorded, Warning: result.Warning, Reason: result.Reason}, nil
}

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
