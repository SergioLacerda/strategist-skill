package install

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/i18n"
)

// promptSlots collects all slot providers. Execution defaults to the certified
// internal Sniper binding, while still exposing the same explicit Ranked versus
// Custom choice used by the other roles.
//
// The discovery/refinement option lists are driven by explicit role affinity
// (compatibleProviderOptions), not a hardcoded slice. Handoff production is a
// fixed role checkpoint, so a weapon is not hidden merely because its adapter
// does not declare the handoff schema.
func promptSlots(p Prompter, b i18n.WizardStrings, catalog pluginCatalog, providerRisk map[string]string, verbose bool) (discovery, refinement, execution, discoveryMode, refinementMode, executionMode string, err error) {
	if verbose {
		fmt.Println(b.HeaderSlots)
	}

	optionSet := newInstallOptionSet(catalog)
	discoveryOptions := optionSet.forSlot(string(domain.SlotDiscovery))
	printExcludedCandidates(discoveryOptions.excluded)
	if len(discoveryOptions.ids) == 0 {
		return "", "", "", "", "", "", fmt.Errorf("wizard: discovery: no compatible weapon for role ranger")
	}
	printRankedRuntimeNote(b, catalog, discoveryOptions.rankedDefault)
	discovery, discoveryMode, err = promptSlotOptions(p, b.PromptDiscovery, discoveryOptions, b.LabelCustomInput, providerRisk, "write_analysis", "discovery")
	if err != nil {
		return "", "", "", "", "", "", err
	}

	refinementOptions := optionSet.forSlot(string(domain.SlotRefinement))
	printExcludedCandidates(refinementOptions.excluded)
	if len(refinementOptions.ids) == 0 {
		return "", "", "", "", "", "", fmt.Errorf("wizard: refinement: no compatible weapon for role archivist")
	}
	printRankedRuntimeNote(b, catalog, refinementOptions.rankedDefault)
	refinement, refinementMode, err = promptSlotOptions(p, b.PromptRefinement, refinementOptions, b.LabelCustomInput, providerRisk, "write_analysis", "refinement")
	if err != nil {
		return "", "", "", "", "", "", err
	}
	execution, executionMode, err = promptExecutionSlot(p, b, catalog, providerRisk)
	if err != nil {
		return "", "", "", "", "", "", err
	}
	return discovery, refinement, execution, discoveryMode, refinementMode, executionMode, nil
}

// rankedRuntimeNote explains the host Node prerequisite of a Ranked option
// whose certified provider declares a runtime, or returns "" when choosing it
// needs nothing from the host.
func rankedRuntimeNote(b i18n.WizardStrings, catalog pluginCatalog, rankedID string) string {
	if rankedID == "" {
		return ""
	}
	provider, ok := findCatalogProviderRef(catalog, rankedID)
	if !ok || domain.NormalizeRankedRuntime(provider.Runtime).Kind == domain.RankedRuntimeNone {
		return ""
	}
	return fmt.Sprintf(b.NoteRankedHostNode, rankedID, domain.MinimumOpenSpecNodeVersion)
}

func printRankedRuntimeNote(b i18n.WizardStrings, catalog pluginCatalog, rankedID string) {
	if note := rankedRuntimeNote(b, catalog, rankedID); note != "" {
		fmt.Println(note)
	}
}

func promptExecutionSlot(p Prompter, b i18n.WizardStrings, catalog pluginCatalog, providerRisk map[string]string) (string, string, error) {
	options := newInstallOptionSet(catalog).forSlot(string(domain.SlotExecution))
	printExcludedCandidates(options.excluded)
	if len(options.ids) == 0 {
		// Older synthetic extractors predate the catalogued internal skill.
		provider, err := promptProvider(p, b.PromptExecution, nativeExecutionProvider, []string{nativeExecutionProvider}, b.LabelCustomInput, providerRisk, "controlled", "execution")
		return provider, domain.SlotBindingModeCustom, err
	}
	return promptSlotOptions(p, b.PromptExecution, options, b.LabelCustomInput, providerRisk, "controlled", "execution")
}

// printExcludedCandidates prints one line per candidate compatibleProviderOptions
// excluded, naming the candidate and its compatibility reasons, so an
// operator seeing a single-option (native-only) prompt can tell an
// intentional exclusion from a bug.
func printExcludedCandidates(excluded []excludedProviderOption) {
	if err := printExcludedCandidatesTo(os.Stdout, excluded); err != nil {
		return
	}
}

func printExcludedCandidatesTo(w io.Writer, excluded []excludedProviderOption) error {
	for _, candidate := range excluded {
		details := make([]string, 0, len(candidate.reasons))
		for _, reason := range candidate.reasons {
			details = append(details, fmt.Sprintf("%s: %s", reason.Code, reason.Detail))
		}
		if _, err := fmt.Fprintf(w, "  %s: excluded — %s\n", candidate.id, strings.Join(details, "; ")); err != nil {
			return fmt.Errorf("write excluded candidate: %w", err)
		}
	}
	return nil
}
