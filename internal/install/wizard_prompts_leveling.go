package install

import (
	"errors"
	"fmt"
	"io"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/i18n"
)

// promptLeveling asks whether model x effort per role is manual (the default)
// or automatic, then prints one notice explaining what the choice means. Manual
// is host passthrough: each role runs with the model and effort the operator
// sets in the host, and Strategist never changes them. Automatic lets the
// LEVELING policy pick them per role by estimated load. It never reads the
// LEVELING policy: only the mode is stored in active.yaml.
//
// Input that ends before this step records automatic, so a piped install script
// written before the step existed keeps the behavior it always had.
func promptLeveling(p Prompter, b i18n.WizardStrings) (domain.LevelingConfig, error) {
	fmt.Println(b.HeaderLeveling)
	mode, err := selectLevelingMode(p, b)
	if err != nil {
		return domain.LevelingConfig{}, err
	}
	if mode == b.LabelLevelingAutomatic {
		fmt.Println(b.NoticeLevelingAutomatic)
		return domain.LevelingConfig{Mode: domain.LevelingModeAutomatic}, nil
	}
	fmt.Println(b.NoticeLevelingManual)
	return domain.LevelingConfig{Mode: domain.LevelingModeManual}, nil
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
