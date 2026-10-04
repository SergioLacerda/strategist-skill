package install

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/i18n"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/setup"
)

// promptIntegration asks the optional provider question. The wizard only
// collects a selection: the shared planner (setup) turns it into a plan, so the
// silent install, the integrations commands and this prompt cannot disagree. The
// endpoint and the data categories are shown before the operator answers.
func promptIntegration(p Prompter, b i18n.WizardStrings, strategistDir string, out io.Writer) (string, error) {
	file, found, err := config.Load(filepath.Join(strategistDir, config.FileName))
	if err != nil {
		return "", fmt.Errorf("wizard: integration: %w", err)
	}
	provider := file.Providers[setup.ProviderName]
	ref, endpoint := firstValue(provider.CredentialRef, setup.DefaultCredentialRef), firstValue(provider.Endpoint, setup.DefaultEndpoint)
	resolves := setup.Resolves(ref, filepath.Dir(strategistDir))
	notice := b.HeaderIntegration + "\n" + fmt.Sprintf(b.NoteIntegrationData, endpoint, strings.Join(setup.DataCategories(), ", "))
	if err := say(out, notice); err != nil {
		return "", err
	}
	choice, err := selectIntegration(p, b, found, resolves)
	if err != nil || choice == "" {
		return "", err
	}
	if !resolves && choice == string(setup.ChoiceEnable) {
		return choice, say(out, b.NoteIntegrationPending)
	}
	return choice, nil
}

func selectIntegration(p Prompter, b i18n.WizardStrings, found, resolves bool) (string, error) {
	options := make([]string, 0, 3)
	for _, choice := range setup.Offered(found) {
		options = append(options, string(choice))
	}
	choice, err := p.Select(b.PromptIntegration, string(setup.DefaultChoice(resolves, found)), options)
	if errors.Is(err, io.EOF) {
		// An input script written before this optional question existed ends
		// here. No answer is no selection, never a default of "enable".
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("wizard: integration: %w", err)
	}
	return choice, nil
}

func say(out io.Writer, text string) error {
	if _, err := fmt.Fprintln(out, text); err != nil {
		return fmt.Errorf("wizard: integration: %w", err)
	}
	return nil
}

func firstValue(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// applyIntegrationChoice is the one place a selection becomes a persisted
// decision, for the wizard and for a silent install alike. An empty selection
// records nothing for a new install and preserves an existing decision.
func applyIntegrationChoice(strategistDir, choice string) error {
	path := filepath.Join(strategistDir, config.FileName)
	file, found, err := config.Load(path)
	if err != nil {
		return fmt.Errorf("install: integration: %w", err)
	}
	ref := firstValue(file.Providers[setup.ProviderName].CredentialRef, setup.DefaultCredentialRef)
	plan, err := setup.Decide(setup.Input{
		Existing: file, Found: found, Choice: setup.Choice(choice),
		CredentialResolves: setup.Resolves(ref, filepath.Dir(strategistDir)),
	})
	if err != nil {
		return fmt.Errorf("install: integration: %w", err)
	}
	if err := setup.Apply(path, plan); err != nil {
		return fmt.Errorf("install: integration: %w", err)
	}
	return nil
}
