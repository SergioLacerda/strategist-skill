package install

import (
	"fmt"
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
	// Manual is host passthrough: the policy is never read.
	if wc.Leveling.HostPassthrough() {
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
	override, found, err := leveling.ReadOverride(path)
	if err != nil {
		return nil, false, fmt.Errorf("read policy file: %w", err)
	}
	return override, found, nil
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

// wizardLevelingSelections returns the selected slot bindings the LEVELING
// policy must resolve.
func wizardLevelingSelections(wc domain.WizardConfig) []wizardLevelingSelection {
	selections := make([]wizardLevelingSelection, 0, 3)
	for _, selection := range []wizardLevelingSelection{
		{wc.DiscoveryProvider, "ranger"},
		{wc.RefinementProvider, "archivist"},
		{wc.ExecutionProvider, "sniper"},
	} {
		if selection.provider != "" {
			selections = append(selections, selection)
		}
	}
	return selections
}

func validateWizardLevelingSelections(policy leveling.Policy, wc domain.WizardConfig) error {
	for _, selection := range wizardLevelingSelections(wc) {
		if _, err := leveling.Suggest(policy, selection.provider, selection.role, leveling.Signals{}); err != nil {
			return fmt.Errorf("resolve LEVELING binding for role %s/provider %s: %w", selection.role, selection.provider, err)
		}
	}
	return nil
}
