package install

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type documentationErrorPrompter struct{}

type documentationFailWriter struct{}

type integrationErrorPrompter struct{ answerPrompter }

func (p *integrationErrorPrompter) Select(title, defaultVal string, options []string) (string, error) {
	if title == i18n.EN.PromptIntegration {
		return "", errors.New("integration prompt failed")
	}
	return p.answerPrompter.Select(title, defaultVal, options)
}

func (documentationFailWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func (documentationErrorPrompter) Select(_, _ string, _ []string) (string, error) {
	return "", errors.New("select failed")
}

func (documentationErrorPrompter) Input(_, _ string) (string, error) {
	return "", errors.New("input failed")
}

func (documentationErrorPrompter) SelectOrInput(_, _ string, _ []string, _ string) (string, error) {
	return "", errors.New("select or input failed")
}

func TestDocumentationRootPromptAndNormalizationErrors(t *testing.T) {
	_, err := promptDocumentationRoot(documentationErrorPrompter{}, i18n.EN.PromptADRRoot, ".analysis/adr", "adr.canonical_path")
	require.ErrorContains(t, err, "adr.canonical_path")

	_, err = normalizeDocumentationRoot(filepath.Join(string(filepath.Separator), "outside", "adr"), "adr.canonical_path", "")
	require.ErrorContains(t, err, "cannot be verified")

	prompter := &documentationRootPrompter{values: []string{"../outside"}}
	_, err = promptDocumentationRootWithin(prompter, i18n.EN.PromptADRRoot, ".analysis/adr", "adr.canonical_path", t.TempDir())
	require.ErrorContains(t, err, "project-relative directory")
}

func TestWriteActiveYAMLRejectsUnsafeDocumentationRootsAndWritesAdditionalRoots(t *testing.T) {
	base := domain.WizardConfig{Mode: "epic", BasePath: ".analysis", UILanguage: "en", DocLanguage: "en", ChatLanguage: "en", CodeLanguage: "en"}
	unsafeADR := base
	unsafeADR.AdrCanonicalPath = "../adr"
	require.ErrorContains(t, writeActiveYAML(t.TempDir(), unsafeADR), "adr.canonical_path")

	unsafeRunbook := base
	unsafeRunbook.RunbookCanonicalPath = "/outside/runbooks"
	require.ErrorContains(t, writeActiveYAML(t.TempDir(), unsafeRunbook), "runbooks.canonical_path")

	configured := base
	configured.DocumentationRoots = []string{".analysis/adr", ".analysis/runbooks"}
	dir := t.TempDir()
	require.NoError(t, writeActiveYAML(dir, configured))
	raw, err := os.ReadFile(filepath.Join(dir, activeYAMLName))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "documentation_roots:")
	assert.Contains(t, string(raw), "- .analysis/adr")
}

func TestWriteActiveYAMLPropagatesInvalidLeveling(t *testing.T) {
	cfg := domain.WizardConfig{Mode: "epic", BasePath: ".analysis", Leveling: domain.LevelingConfig{Mode: "unknown"}}
	require.ErrorContains(t, writeActiveYAML(t.TempDir(), cfg), "leveling_mapping_invalid")
}

func TestWizardCoverageForEmptyActivationAndExecutionCatalog(t *testing.T) {
	lock, err := attachWeaponBindingArtifacts(domain.PluginLockFile{})
	require.NoError(t, err)
	assert.Nil(t, lock.BindingArtifacts)

	provider, mode, err := promptExecutionSlot(&answerPrompter{}, i18n.EN, pluginCatalog{}, nil)
	require.NoError(t, err)
	assert.Equal(t, nativeExecutionProvider, provider)
	assert.Equal(t, domain.SlotBindingModeCustom, mode)

	err = validateWizardActivationInputs(minimalExtractor{}, domain.WizardConfig{}, pluginOnboardingPlan{}, "", false)
	require.ErrorContains(t, err, "invalid install plan")

	_, _, _, _, _, _, err = promptSlots(&answerPrompter{}, i18n.EN, versionedRangerCatalog(), knownProviderRisk, false)
	require.ErrorContains(t, err, "refinement: no compatible weapon")

	err = printExcludedCandidatesTo(documentationFailWriter{}, []excludedProviderOption{{id: "excluded", reasons: []domain.CompatibilityReason{{Code: "reason", Detail: "detail"}}}})
	require.ErrorContains(t, err, "write excluded candidate")

	catalog, _, wc := defaultOnboardingPlan(t)
	badRoot := filepath.Join(t.TempDir(), "blocked")
	require.NoError(t, os.WriteFile(badRoot, nil, 0o644))
	_, err = applyWizardActivation(defaultsExtractor{}, catalog, wc, badRoot, pluginOnboardingPlan{})
	require.ErrorContains(t, err, "activate role/provider migration")

	invalidBinding := domain.SlotBinding{
		TaxonomyVersion: "invalid", Slot: "discovery", InstalledInstanceID: "instance", Role: "ranger",
		WeaponVersion: "1.0.0", WeaponDigest: "weapon", BindingDigest: "binding", RuntimeKind: "host",
		ConnectorID: "connector", Entrypoint: "entrypoint", Origin: "custom",
	}
	_, err = attachWeaponBindingArtifacts(domain.PluginLockFile{Bindings: []domain.SlotBinding{invalidBinding}})
	require.ErrorContains(t, err, "build Weapon Binding Artifacts")
}

func TestRunWizardPropagatesIntegrationPromptFailure(t *testing.T) {
	_, err := runWizard(context.Background(), &integrationErrorPrompter{}, defaultsExtractor{}, "", false)
	require.ErrorContains(t, err, "wizard: integration")
}

func TestValidateAndActivatePluginPlanPropagatesPlanningFailure(t *testing.T) {
	_, err := validateAndActivatePluginPlan(minimalExtractor{}, pluginCatalog{}, nil, domain.WizardConfig{}, "", false)
	require.ErrorContains(t, err, "plugin onboarding plan")
}
