package install

import (
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runLevelingPrompt(t *testing.T, lang, input string) domain.LevelingConfig {
	t.Helper()
	cfg, err := promptLeveling(NewTextPrompter(strings.NewReader(input)), i18n.BundleFor(lang))
	require.NoError(t, err)
	return cfg
}

func TestPromptLevelingDefaultsToManualForAllRoles(t *testing.T) {
	// Enter on every question: manual, apply to all, no model, default effort.
	cfg := runLevelingPrompt(t, "en", "\n\n\n\n")
	require.Equal(t, domain.LevelingModeManual, cfg.Mode)
	require.Len(t, cfg.Roles, len(domain.LevelingRoleIDs()))
	for _, role := range domain.LevelingRoleIDs() {
		assert.Equal(t, domain.LevelingRoleChoice{Effort: defaultManualEffort}, cfg.Roles[role], role)
	}
	require.NoError(t, cfg.Validate())
}

func TestPromptLevelingAutomaticIsExplicitOptIn(t *testing.T) {
	cfg := runLevelingPrompt(t, "en", "automatic\n")
	assert.Equal(t, domain.LevelingConfig{Mode: domain.LevelingModeAutomatic}, cfg)
}

func TestPromptLevelingExhaustedInputKeepsAutomatic(t *testing.T) {
	// A piped install script written before this step existed ends before the
	// new answer; it must keep working with the behavior it always had.
	cfg := runLevelingPrompt(t, "en", "")
	assert.Equal(t, domain.LevelingModeAutomatic, cfg.Mode)
	assert.Empty(t, cfg.Roles)
}

func TestPromptLevelingManualAppliesOneChoiceToAllRoles(t *testing.T) {
	cfg := runLevelingPrompt(t, "en", "manual\nall\nSonnet\nhigh\n")
	require.Equal(t, domain.LevelingModeManual, cfg.Mode)
	require.Len(t, cfg.Roles, len(domain.LevelingRoleIDs()))
	for _, role := range domain.LevelingRoleIDs() {
		assert.Equal(t, domain.LevelingRoleChoice{Model: "Sonnet", Effort: "high"}, cfg.Roles[role], role)
	}
	require.NoError(t, cfg.Validate())
}

func TestPromptLevelingManualPerRole(t *testing.T) {
	// scout: blank model, effort low; ranger; archivist; sniper.
	input := "manual\nper role\n\nlow\nSonnet\nhigh\nOpus\nmedium\nHaiku\nlow\n"
	cfg := runLevelingPrompt(t, "en", input)
	assert.Equal(t, map[string]domain.LevelingRoleChoice{
		"scout":     {Effort: "low"},
		"ranger":    {Model: "Sonnet", Effort: "high"},
		"archivist": {Model: "Opus", Effort: "medium"},
		"sniper":    {Model: "Haiku", Effort: "low"},
	}, cfg.Roles)
	require.NoError(t, cfg.Validate())
}

func TestPromptLevelingIsLocalized(t *testing.T) {
	cfg := runLevelingPrompt(t, "pt-BR", "manual\ntodos\nOpus\nxhigh\n")
	assert.Equal(t, domain.LevelingModeManual, cfg.Mode)
	assert.Equal(t, domain.LevelingRoleChoice{Model: "Opus", Effort: "xhigh"}, cfg.Roles["ranger"])

	auto := runLevelingPrompt(t, "pt-BR", "automático\n")
	assert.Equal(t, domain.LevelingModeAutomatic, auto.Mode)
	assert.NotEmpty(t, i18n.EN.LabelLevelingManual+i18n.EN.LabelLevelingAll)
}

func TestPromptLevelingAllStringsPresentInBothBundles(t *testing.T) {
	for _, bundle := range []i18n.WizardStrings{i18n.EN, i18n.PT} {
		for name, value := range map[string]string{
			"HeaderLeveling": bundle.HeaderLeveling, "PromptLevelingMode": bundle.PromptLevelingMode,
			"LabelLevelingAutomatic": bundle.LabelLevelingAutomatic, "LabelLevelingManual": bundle.LabelLevelingManual,
			"PromptLevelingScope": bundle.PromptLevelingScope, "LabelLevelingAll": bundle.LabelLevelingAll,
			"LabelLevelingPerRole": bundle.LabelLevelingPerRole, "PromptLevelingModel": bundle.PromptLevelingModel,
			"PromptLevelingEffort": bundle.PromptLevelingEffort, "SummaryLeveling": bundle.SummaryLeveling,
		} {
			assert.NotEmpty(t, value, name)
		}
	}
}

func TestPromptLevelingInputEndingMidStepKeepsAutomatic(t *testing.T) {
	for _, input := range []string{"\n", "manual\n", "manual\nper role\nSonnet\nhigh\n"} {
		cfg := runLevelingPrompt(t, "en", input)
		assert.Equal(t, domain.LevelingConfig{Mode: domain.LevelingModeAutomatic}, cfg, "input %q", input)
	}
}
