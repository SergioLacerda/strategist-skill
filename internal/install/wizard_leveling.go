package install

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
)

const (
	levelingPolicyPath        = "leveling.yaml"
	levelingCompatibilityPath = ".leveling-compat.yaml"
)

// validateWizardLeveling checks the effective policy and every selected slot
// before role/provider activation. Optional extractors keep white-box callers
// source-compatible while production passes the embedded extractor explicitly.
func validateWizardLeveling(strategistDir string, wc domain.WizardConfig, extractors ...domain.FileExtractor) error {
	if strategistDir == "" {
		return nil
	}
	// Roles the operator set manually need nothing from the policy; when every
	// selected slot is manual the policy is not even read.
	if needPolicy, total := wizardLevelingSelections(wc); total > 0 && len(needPolicy) == 0 {
		return nil
	}
	policy, found, err := loadWizardLevelingPolicy(strategistDir, extractors...)
	if err != nil {
		return fmt.Errorf("load LEVELING policy: %w", err)
	}
	if !found {
		return nil
	}
	return validateWizardLevelingSelections(policy, wc)
}

func loadWizardLevelingPolicy(strategistDir string, extractors ...domain.FileExtractor) (leveling.Policy, bool, error) {
	path := filepath.Join(strategistDir, levelingPolicyPath)
	override, found, err := readWizardLevelingOverride(path)
	if err != nil {
		return leveling.Policy{}, found, err
	}
	if !found && len(extractors) == 0 {
		return legacyWizardLevelingPolicy(strategistDir)
	}
	defaults, err := wizardLevelingDefaults(override, extractors...)
	if err != nil {
		return leveling.Policy{}, false, err
	}
	effective, _, err := leveling.LoadAuthorized(strategistDir, defaults, override, path)
	if err != nil {
		return leveling.Policy{}, false, fmt.Errorf("load authorized policy: %w", err)
	}
	return effective.Policy, true, nil
}

func legacyWizardLevelingPolicy(strategistDir string) (leveling.Policy, bool, error) {
	if err := leveling.ValidateLegacyCompatibility(strategistDir); err != nil {
		return leveling.Policy{}, false, fmt.Errorf("validate legacy compatibility: %w", err)
	}
	return leveling.Policy{}, false, nil
}

func readWizardLevelingOverride(path string) ([]byte, bool, error) {
	override, err := os.ReadFile(path) //nolint:gosec // fixed path below the selected .strategist root
	if err == nil {
		return override, true, nil
	}
	if !os.IsNotExist(err) {
		return nil, false, fmt.Errorf("read policy file: %w", err)
	}
	return nil, false, nil
}

func wizardLevelingDefaults(override []byte, extractors ...domain.FileExtractor) ([]byte, error) {
	if len(extractors) > 0 && extractors[0] != nil {
		defaults, err := extractors[0].ReadFile(levelingPolicyPath)
		if err != nil {
			return nil, fmt.Errorf("leveling_policy_stale: read embedded defaults: %w", err)
		}
		if len(defaults) == 0 {
			return nil, fmt.Errorf("leveling_policy_stale: embedded leveling.yaml is empty")
		}
		return defaults, nil
	}
	return override, nil
}

type wizardLevelingSelection struct {
	provider string
	role     string
}

// wizardLevelingSelections returns the slot bindings that still need the
// LEVELING policy: those whose role is not fully set manually. total counts the
// selected slots so the caller can tell "nothing selected" from "all manual".
func wizardLevelingSelections(wc domain.WizardConfig) (needPolicy []wizardLevelingSelection, total int) {
	for _, selection := range []wizardLevelingSelection{
		{wc.DiscoveryProvider, "ranger"},
		{wc.RefinementProvider, "archivist"},
		{wc.ExecutionProvider, "sniper"},
	} {
		if selection.provider == "" {
			continue
		}
		total++
		if choice, manual := wc.Leveling.Choice(selection.role); manual && choice.Complete() {
			continue
		}
		needPolicy = append(needPolicy, selection)
	}
	return needPolicy, total
}

func validateWizardLevelingSelections(policy leveling.Policy, wc domain.WizardConfig) error {
	selections, _ := wizardLevelingSelections(wc)
	for _, selection := range selections {
		if _, err := leveling.Suggest(policy, selection.provider, selection.role, leveling.Signals{}); err != nil {
			return fmt.Errorf("resolve LEVELING binding for role %s/provider %s: %w", selection.role, selection.provider, err)
		}
	}
	return nil
}
