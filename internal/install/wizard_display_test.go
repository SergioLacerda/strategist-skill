package install

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureWizardStdout runs fn with os.Stdout redirected. It is only used by
// tests that do not call t.Parallel, so no other test prints while it is swapped.
func captureWizardStdout(t *testing.T, fn func()) string {
	t.Helper()
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	original := os.Stdout
	os.Stdout = writer
	fn()
	os.Stdout = original
	require.NoError(t, writer.Close())
	var out bytes.Buffer
	_, err = io.Copy(&out, reader)
	require.NoError(t, err)
	return out.String()
}

func wizardInputFor(lang string) string {
	return lang + "\n" + lang + "\n" + lang + "\nen\nepic\n/workspace\nbrainstorming\narchivist\nfixture-provider\n\n"
}

func runDisplayWizard(t *testing.T, lang string, verbose bool) string {
	t.Helper()
	return captureWizardStdout(t, func() {
		_, err := runWizard(context.Background(), NewTextPrompter(strings.NewReader(wizardInputFor(lang))), minimalExtractor{}, "", verbose)
		require.NoError(t, err)
	})
}

func TestWizardHidesTheSectionHeadersAndMigrationPreviewByDefault(t *testing.T) {
	for lang, bundle := range map[string]i18n.WizardStrings{"en": i18n.EN, "pt-BR": i18n.PT} {
		out := runDisplayWizard(t, lang, false)

		assert.NotContains(t, out, strings.TrimSpace(bundle.HeaderSlots), lang)
		assert.NotContains(t, out, strings.TrimSpace(bundle.HeaderChest), lang)
		assert.NotContains(t, out, "role/provider migration preview", lang)
		assert.Contains(t, out, strings.TrimSpace(bundle.PromptDiscovery), "%s: the prompts are still shown", lang)
		assert.Contains(t, out, strings.TrimSpace(bundle.PromptChestPath), lang)
	}
}

func TestWizardShowsTheSectionHeadersAndMigrationPreviewWhenVerbose(t *testing.T) {
	for lang, bundle := range map[string]i18n.WizardStrings{"en": i18n.EN, "pt-BR": i18n.PT} {
		out := runDisplayWizard(t, lang, true)

		assert.Contains(t, out, strings.TrimSpace(bundle.HeaderSlots), lang)
		assert.Contains(t, out, strings.TrimSpace(bundle.HeaderChest), lang)
		assert.Contains(t, out, "role/provider migration preview", lang)
	}
}

func TestWizardPortugueseHeadersMatchTheRequestedStrings(t *testing.T) {
	out := runDisplayWizard(t, "pt-BR", true)

	assert.Contains(t, out, "Plugins de slot - qual skill preenche cada papel da missão")
	assert.Contains(t, out, "Baú do tesouro — base de conhecimento offline opcional para todos os slots:")
}
