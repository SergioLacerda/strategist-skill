// Package levelingapp orchestrates LEVELING resolution and ledger recording.
package levelingapp

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	leveling "github.com/SergioLacerda/strategist-skill/internal/tools/leveling"
)

const (
	defaultLevelingMaxRecords = 2000
	levelingLedgerRotateBytes = 512 * 1024
)

// LevelingInput is the application-owned input for resolving and recording a
// LEVELING result. CLI flag parsing and telemetry rendering remain adapters.
type LevelingInput struct {
	Role, Provider, Mission, Run, HostModel, HostEffort string
	Ambiguity, Risk, Scope, Evidence, Reason            string
}

// LevelingResult is the stable application result consumed by CLI adapters.
type LevelingResult struct {
	Level    leveling.Level
	Reused   bool
	Recorded bool
	Warning  string
	Reason   string
}

// ResolveLevel resolves, records, and rotates one role-level ledger entry.
// Policy and registry loading are supplied by the composition root.
func ResolveLevel(reg domain.RoleRegistry, load leveling.PolicyLoader, cfg domain.LevelingConfig, ledgerPath string, input LevelingInput, maxRecords int) (LevelingResult, error) {
	if maxRecords <= 0 {
		maxRecords = defaultLevelingMaxRecords
	}
	if result, reused, err := reuseLevel(ledgerPath, input); err != nil {
		return LevelingResult{}, err
	} else if reused {
		return result, nil
	}
	result := resolveLevel(reg, load, cfg, input)
	if err := recordLevel(ledgerPath, input, &result, maxRecords); err != nil {
		return LevelingResult{}, err
	}
	return result, nil
}

func reuseLevel(ledgerPath string, input LevelingInput) (LevelingResult, bool, error) {
	if input.Mission == "" || input.Reason != "" {
		return LevelingResult{}, false, nil
	}
	record, ok, err := leveling.LatestRunRecord(ledgerPath, input.Mission, input.Role, input.Run)
	if err != nil {
		return LevelingResult{}, false, fmt.Errorf("leveling: read role level ledger: %w", err)
	}
	if !ok || record.Unknown() {
		return LevelingResult{}, false, nil
	}
	return LevelingResult{Level: record.Level, Reused: true}, true, nil
}

func resolveLevel(reg domain.RoleRegistry, load leveling.PolicyLoader, cfg domain.LevelingConfig, input LevelingInput) LevelingResult {
	provider := input.Provider
	if cfg.HostPassthrough() {
		provider = ""
	}
	host, rejected := leveling.SanitizeHost(leveling.Host{Model: input.HostModel, Effort: input.HostEffort})
	signals := leveling.Signals{Ambiguity: input.Ambiguity, Risk: input.Risk, Scope: input.Scope, Evidence: input.Evidence}
	resolve := func() (leveling.Level, error) {
		return leveling.ResolveLevelLazy(load, provider, reg.PolicyRole(input.Role), signals, host)
	}
	if provider == "" && !cfg.HostPassthrough() {
		resolve = func() (leveling.Level, error) {
			return leveling.ResolveLevelInferred(load, reg.PolicyRole(input.Role), signals, host)
		}
	}
	level, err := resolve()
	if err != nil {
		return LevelingResult{Level: leveling.Level{Role: leveling.NormalizeRole(input.Role)}, Warning: joinLevelWarnings(append(rejected, err.Error()))}
	}
	level.Role = leveling.NormalizeRole(input.Role)
	return LevelingResult{Level: level, Warning: joinLevelWarnings(rejected)}
}

func joinLevelWarnings(warnings []string) string { return strings.Join(warnings, "; ") }

func recordLevel(ledgerPath string, input LevelingInput, result *LevelingResult, maxRecords int) error {
	if input.Mission == "" {
		return nil
	}
	ok, err := shouldRecordLevel(ledgerPath, input, result.Level)
	if err != nil || !ok {
		return err
	}
	result.Recorded, result.Reason = true, input.Reason
	if err := leveling.AppendRecord(ledgerPath, leveling.Record{MissionID: input.Mission, Run: input.Run, Level: result.Level, Reason: input.Reason}); err != nil {
		return fmt.Errorf("leveling: record role level: %w", err)
	}
	if _, err := leveling.RotateLedgerIfLarge(ledgerPath, levelingLedgerRotateBytes, maxRecords); err != nil {
		result.Warning = err.Error()
	}
	return nil
}

func shouldRecordLevel(ledgerPath string, input LevelingInput, level leveling.Level) (bool, error) {
	if leveling.NormalizeRole(input.Role) == "gate" {
		return false, nil
	}
	if !level.Unknown() || input.Reason != "" {
		return true, nil
	}
	latest, found, err := leveling.LatestRunRecord(ledgerPath, input.Mission, input.Role, input.Run)
	if err != nil {
		return false, fmt.Errorf("leveling: read role level ledger: %w", err)
	}
	return !found || !latest.Unknown(), nil
}
