package install

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/i18n"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/stretchr/testify/require"
)

// answerPrompter answers Select prompts from a fixed script and records what it
// was asked, so a test can assert both the question and its default.
type answerPrompter struct {
	answer   string
	titles   []string
	defaults []string
	options  [][]string
}

func (a *answerPrompter) Select(title, defaultVal string, options []string) (string, error) {
	a.titles, a.defaults, a.options = append(a.titles, title), append(a.defaults, defaultVal), append(a.options, options)
	if a.answer != "" {
		return a.answer, nil
	}
	return defaultVal, nil
}
func (a *answerPrompter) Input(_, defaultVal string) (string, error) { return defaultVal, nil }
func (a *answerPrompter) SelectOrInput(_, defaultVal string, _ []string, _ string) (string, error) {
	return defaultVal, nil
}

func strategistRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), ".strategist")
	require.NoError(t, os.MkdirAll(root, 0o755))
	return root
}

func TestWizardAsksAboutTheIntegrationAndShowsTheEndpointAndDataCategories(t *testing.T) {
	root := strategistRoot(t)
	prompter := &answerPrompter{}
	var out bytes.Buffer

	choice, err := promptIntegration(prompter, i18n.EN, root, &out)
	require.NoError(t, err)
	require.Len(t, prompter.titles, 1, "AC-01: the interactive install asks, with no additional skill required")
	require.Equal(t, i18n.EN.PromptIntegration, prompter.titles[0])
	require.Equal(t, []string{"enable", "disable"}, prompter.options[0], "keep needs a prior decision")
	require.Equal(t, "disable", prompter.defaults[0], "no credential resolves, so the default is no")
	require.Equal(t, "disable", choice)
	require.Contains(t, out.String(), "https://api.typesafe.ai/v1/systemone")
	require.Contains(t, out.String(), "implementation_plan", "the data categories are shown before consent")
}

func TestWizardDefaultsToYesWhenTheDotenvCredentialResolves(t *testing.T) {
	root := strategistRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(root), ".env"), []byte("TYPESAFE_API_KEY=sk-wizard\n"), 0o600))
	prompter := &answerPrompter{}

	choice, err := promptIntegration(prompter, i18n.EN, root, &bytes.Buffer{})
	require.NoError(t, err)
	require.Equal(t, "enable", prompter.defaults[0])
	require.Equal(t, "enable", choice)
}

func TestWizardOffersKeepAndDefaultsToItWhenADecisionExists(t *testing.T) {
	root := strategistRoot(t)
	require.NoError(t, applyIntegrationChoice(root, "enable"))
	prompter := &answerPrompter{}

	choice, err := promptIntegration(prompter, i18n.PT, root, &bytes.Buffer{})
	require.NoError(t, err)
	require.Equal(t, []string{"enable", "disable", "keep"}, prompter.options[0])
	require.Equal(t, "keep", choice)
	require.Equal(t, i18n.PT.PromptIntegration, prompter.titles[0], "the question is localized")
}

func TestWizardWarnsThatAnEnabledProviderWithoutACredentialIsPending(t *testing.T) {
	root := strategistRoot(t)
	var out bytes.Buffer
	_, err := promptIntegration(&answerPrompter{answer: "enable"}, i18n.EN, root, &out)
	require.NoError(t, err)
	require.Contains(t, out.String(), "pending", "WIZ-07: the missing credential is saved as pending, not silently ignored")
}

func TestOneSelectionYieldsTheSameFileInWizardAndSilentInstall(t *testing.T) {
	wizardRoot, silentRoot := strategistRoot(t), strategistRoot(t)

	choice, err := promptIntegration(&answerPrompter{answer: "enable"}, i18n.EN, wizardRoot, &bytes.Buffer{})
	require.NoError(t, err)
	require.NoError(t, applyIntegrationChoice(wizardRoot, choice))
	require.NoError(t, applyIntegrationChoice(silentRoot, "enable"))

	wizard, err := os.ReadFile(filepath.Join(wizardRoot, config.FileName))
	require.NoError(t, err)
	silent, err := os.ReadFile(filepath.Join(silentRoot, config.FileName))
	require.NoError(t, err)
	require.Equal(t, string(wizard), string(silent), "one selection, one plan, whatever the adapter")
}

func TestSilentInstallWithoutASelectionLeavesTheIntegrationDisabled(t *testing.T) {
	root := strategistRoot(t)
	t.Setenv("TYPESAFE_API_KEY", "sk-present-but-not-consent")
	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(root), ".env"), []byte("TYPESAFE_API_KEY=sk-x\n"), 0o600))

	require.NoError(t, applyIntegrationChoice(root, ""))
	require.NoFileExists(t, filepath.Join(root, config.FileName), "AC-03 and AC-16: a variable or .env alone never enables, and a new silent install records nothing")
}

func TestSilentUpgradePreservesAnExistingDecision(t *testing.T) {
	root := strategistRoot(t)
	require.NoError(t, applyIntegrationChoice(root, "enable"))
	before, err := os.ReadFile(filepath.Join(root, config.FileName))
	require.NoError(t, err)

	require.NoError(t, applyIntegrationChoice(root, ""))
	require.NoError(t, applyIntegrationChoice(root, "keep"))
	after, err := os.ReadFile(filepath.Join(root, config.FileName))
	require.NoError(t, err)
	require.Equal(t, string(before), string(after))
}

func TestAnUnknownSilentSelectionIsRejected(t *testing.T) {
	require.Error(t, applyIntegrationChoice(strategistRoot(t), "maybe"))
}

type eofPrompter struct{ answerPrompter }

func (eofPrompter) Select(string, string, []string) (string, error) {
	return "", fmt.Errorf("read: %w", io.EOF)
}

func TestAnExhaustedInputScriptMeansNoSelectionNeverEnable(t *testing.T) {
	root := strategistRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(root), ".env"), []byte("TYPESAFE_API_KEY=sk-x\n"), 0o600))

	choice, err := promptIntegration(&eofPrompter{}, i18n.EN, root, &bytes.Buffer{})
	require.NoError(t, err, "scripts written before the question existed keep working")
	require.Empty(t, choice)
	require.NoError(t, applyIntegrationChoice(root, choice))
	require.NoFileExists(t, filepath.Join(root, config.FileName))
}

func TestWizardNeverEchoesOrStoresTheCredential(t *testing.T) {
	const secret = "sk-wizard-scan-0099887766"
	root := strategistRoot(t)
	workspace := filepath.Dir(root)
	require.NoError(t, os.WriteFile(filepath.Join(workspace, ".env"), []byte("TYPESAFE_API_KEY="+secret+"\n"), 0o600))
	var out bytes.Buffer

	choice, err := promptIntegration(&answerPrompter{answer: "enable"}, i18n.EN, root, &out)
	require.NoError(t, err)
	require.NoError(t, applyIntegrationChoice(root, choice))
	require.NotContains(t, out.String(), secret, "the wizard output")

	require.NoError(t, filepath.WalkDir(workspace, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || entry.Name() == ".env" {
			return walkErr
		}
		data, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		require.NotContains(t, string(data), secret, path)
		return nil
	}))
	require.Empty(t, os.Getenv("TYPESAFE_API_KEY"), "the .env value is never exported to the process")
}
