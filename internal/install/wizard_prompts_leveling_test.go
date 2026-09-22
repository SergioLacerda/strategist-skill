package install

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runLevelingPrompt answers the LEVELING step with input and returns the
// resulting config plus everything the step printed.
func runLevelingPrompt(t *testing.T, lang, input string) (domain.LevelingConfig, string) {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	prev := os.Stdout
	os.Stdout = w
	cfg, promptErr := promptLeveling(NewTextPrompter(strings.NewReader(input)), i18n.BundleFor(lang))
	os.Stdout = prev
	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, promptErr)
	return cfg, string(out)
}

func TestPromptLevelingDefaultsToManualAndOnlyNotifies(t *testing.T) {
	// Enter on the mode question selects manual; nothing else is asked, so the
	// remaining input is never read.
	cfg, out := runLevelingPrompt(t, "en", "\nleftover\n")
	assert.Equal(t, domain.LevelingConfig{Mode: domain.LevelingModeManual}, cfg)
	require.NoError(t, cfg.Validate())
	assert.Equal(t, 1, strings.Count(out, i18n.EN.NoticeLevelingManual), "manual notice printed once")
	assert.NotContains(t, out, i18n.EN.NoticeLevelingAutomatic)
}

func TestPromptLevelingAutomaticIsExplicitOptIn(t *testing.T) {
	cfg, out := runLevelingPrompt(t, "en", "automatic\n")
	assert.Equal(t, domain.LevelingConfig{Mode: domain.LevelingModeAutomatic}, cfg)
	assert.Equal(t, 1, strings.Count(out, i18n.EN.NoticeLevelingAutomatic), "automatic notice printed once")
	assert.NotContains(t, out, i18n.EN.NoticeLevelingManual)
}

func TestPromptLevelingExhaustedInputKeepsAutomatic(t *testing.T) {
	// A piped install script written before this step existed ends before the
	// new answer; it must keep working with the behavior it always had.
	cfg, out := runLevelingPrompt(t, "en", "")
	assert.Equal(t, domain.LevelingConfig{Mode: domain.LevelingModeAutomatic}, cfg)
	assert.Contains(t, out, i18n.EN.NoticeLevelingAutomatic)
}

func TestPromptLevelingIsLocalized(t *testing.T) {
	cfg, out := runLevelingPrompt(t, "pt-BR", "manual\n")
	assert.Equal(t, domain.LevelingModeManual, cfg.Mode)
	assert.Contains(t, out, i18n.PT.NoticeLevelingManual)

	auto, out := runLevelingPrompt(t, "pt-BR", "automático\n")
	assert.Equal(t, domain.LevelingModeAutomatic, auto.Mode)
	assert.Contains(t, out, i18n.PT.NoticeLevelingAutomatic)
}

func TestPromptLevelingAllStringsPresentInBothBundles(t *testing.T) {
	for _, bundle := range []i18n.WizardStrings{i18n.EN, i18n.PT} {
		for name, value := range map[string]string{
			"HeaderLeveling": bundle.HeaderLeveling, "PromptLevelingMode": bundle.PromptLevelingMode,
			"LabelLevelingAutomatic": bundle.LabelLevelingAutomatic, "LabelLevelingManual": bundle.LabelLevelingManual,
			"NoticeLevelingManual": bundle.NoticeLevelingManual, "NoticeLevelingAutomatic": bundle.NoticeLevelingAutomatic,
		} {
			assert.NotEmpty(t, value, name)
		}
	}
}
