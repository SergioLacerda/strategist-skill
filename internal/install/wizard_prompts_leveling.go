package install

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/i18n"
	"github.com/SergioLacerda/strategist-skill/internal/leveling"
)

const defaultManualEffort = "medium"

// promptLeveling asks whether model x effort per role is decided manually (the
// default) or automatically by the LEVELING policy, and for manual collects one
// choice for every role or one per role. Accepting every default therefore
// yields a manual choice for all roles with effort "medium" and no model, so the
// model still comes from the host or, on demand, the policy. It never reads the LEVELING policy:
// the answer is stored in active.yaml and read on demand at runtime.
//
// Input that ends before or inside this step records automatic (no complete
// manual answer exists to store), so a piped install script written before the
// step existed keeps the behavior it always had.
func promptLeveling(p Prompter, b i18n.WizardStrings) (domain.LevelingConfig, error) {
	fmt.Println(b.HeaderLeveling)
	mode, err := selectLevelingMode(p, b)
	if err != nil {
		return domain.LevelingConfig{}, err
	}
	if mode == b.LabelLevelingAutomatic {
		return domain.LevelingConfig{Mode: domain.LevelingModeAutomatic}, nil
	}
	roles, err := promptManualLevelingRoles(p, b)
	if errors.Is(err, io.EOF) {
		return domain.LevelingConfig{Mode: domain.LevelingModeAutomatic}, nil
	}
	if err != nil {
		return domain.LevelingConfig{}, err
	}
	return finalizeManualLeveling(roles, b)
}

func selectLevelingMode(p Prompter, b i18n.WizardStrings) (string, error) {
	mode, err := p.Select(b.PromptLevelingMode, b.LabelLevelingManual, []string{b.LabelLevelingManual, b.LabelLevelingAutomatic})
	if errors.Is(err, io.EOF) {
		return b.LabelLevelingAutomatic, nil
	}
	if err != nil {
		return "", fmt.Errorf("wizard: leveling: %w", err)
	}
	return mode, nil
}

func finalizeManualLeveling(roles map[string]domain.LevelingRoleChoice, b i18n.WizardStrings) (domain.LevelingConfig, error) {
	cfg := domain.LevelingConfig{Mode: domain.LevelingModeManual, Roles: roles}
	if err := cfg.Validate(); err != nil {
		return domain.LevelingConfig{}, fmt.Errorf("wizard: leveling: %w", err)
	}
	for _, role := range domain.LevelingRoleIDs() {
		choice := roles[role]
		fmt.Printf(b.SummaryLeveling+"\n", role, leveling.Level{Model: choice.Model, Effort: choice.Effort}.Label())
	}
	return cfg, nil
}

func promptManualLevelingRoles(p Prompter, b i18n.WizardStrings) (map[string]domain.LevelingRoleChoice, error) {
	scope, err := p.Select(b.PromptLevelingScope, b.LabelLevelingAll, []string{b.LabelLevelingAll, b.LabelLevelingPerRole})
	if err != nil {
		return nil, fmt.Errorf("wizard: leveling scope: %w", err)
	}
	roles := make(map[string]domain.LevelingRoleChoice, len(domain.LevelingRoleIDs()))
	if scope == b.LabelLevelingAll {
		return applyChoiceToRoles(p, b, roles)
	}
	return promptEachRole(p, b, roles)
}

func applyChoiceToRoles(p Prompter, b i18n.WizardStrings, roles map[string]domain.LevelingRoleChoice) (map[string]domain.LevelingRoleChoice, error) {
	choice, err := promptLevelingChoice(p, b, "*")
	if err != nil {
		return nil, err
	}
	for _, role := range domain.LevelingRoleIDs() {
		roles[role] = choice
	}
	return roles, nil
}

func promptEachRole(p Prompter, b i18n.WizardStrings, roles map[string]domain.LevelingRoleChoice) (map[string]domain.LevelingRoleChoice, error) {
	for _, role := range domain.LevelingRoleIDs() {
		choice, err := promptLevelingChoice(p, b, role)
		if err != nil {
			return nil, err
		}
		roles[role] = choice
	}
	return roles, nil
}

func promptLevelingChoice(p Prompter, b i18n.WizardStrings, role string) (domain.LevelingRoleChoice, error) {
	model, err := p.Input(fmt.Sprintf(b.PromptLevelingModel, role), "")
	if err != nil {
		return domain.LevelingRoleChoice{}, fmt.Errorf("wizard: leveling model for %s: %w", role, err)
	}
	effort, err := p.Select(fmt.Sprintf(b.PromptLevelingEffort, role), defaultManualEffort, domain.LevelingEffortTiers)
	if err != nil {
		return domain.LevelingRoleChoice{}, fmt.Errorf("wizard: leveling effort for %s: %w", role, err)
	}
	return domain.LevelingRoleChoice{Model: strings.TrimSpace(model), Effort: effort}, nil
}
