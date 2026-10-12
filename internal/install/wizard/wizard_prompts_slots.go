package wizard

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/i18n"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
)

// PromptSlots collects all slot providers. Execution defaults to the certified
// internal Sniper binding, while still exposing the same explicit Ranked versus
// Custom choice used by the other roles.
//
// The discovery/refinement option lists are driven by explicit role affinity
// (compatibleProviderOptions), not a hardcoded slice. Handoff production is a
// fixed role checkpoint, so a weapon is not hidden merely because its adapter
// does not declare the handoff schema.
func PromptSlots(p Prompter, b i18n.WizardStrings, catalog plugincatalog.PluginCatalog, providerRisk map[string]string, verbose bool, defaults Defaults) (discovery, refinement, execution, discoveryMode, refinementMode, executionMode string, err error) {
	if verbose {
		fmt.Println(b.HeaderSlots)
	}

	optionSet := plugincatalog.NewInstallOptionSet(catalog)
	discoveryOptions := defaults.applyToSlot(optionSet.ForSlot(string(domain.SlotDiscovery)), string(domain.SlotDiscovery))
	printExcludedCandidates(discoveryOptions.Excluded)
	if len(discoveryOptions.IDs) == 0 {
		return "", "", "", "", "", "", fmt.Errorf("wizard: discovery: no compatible weapon for role ranger")
	}
	printRankedRuntimeNote(b, catalog, discoveryOptions.RankedDefault)
	discovery, discoveryMode, err = PromptSlotOptions(p, b.PromptDiscovery, discoveryOptions, b.LabelCustomInput, providerRisk, names.RiskWriteAnalysis, string(domain.SlotDiscovery))
	if err != nil {
		return "", "", "", "", "", "", err
	}

	refinementOptions := defaults.applyToSlot(optionSet.ForSlot(string(domain.SlotRefinement)), string(domain.SlotRefinement))
	printExcludedCandidates(refinementOptions.Excluded)
	if len(refinementOptions.IDs) == 0 {
		return "", "", "", "", "", "", fmt.Errorf("wizard: refinement: no compatible weapon for role archivist")
	}
	printRankedRuntimeNote(b, catalog, refinementOptions.RankedDefault)
	refinement, refinementMode, err = PromptSlotOptions(p, b.PromptRefinement, refinementOptions, b.LabelCustomInput, providerRisk, names.RiskWriteAnalysis, string(domain.SlotRefinement))
	if err != nil {
		return "", "", "", "", "", "", err
	}
	execution, executionMode, err = PromptExecutionSlot(p, b, catalog, providerRisk, defaults)
	if err != nil {
		return "", "", "", "", "", "", err
	}
	return discovery, refinement, execution, discoveryMode, refinementMode, executionMode, nil
}

// rankedRuntimeNote explains the host Node prerequisite of a Ranked option
// whose certified provider declares a runtime, or returns "" when choosing it
// needs nothing from the host.
func rankedRuntimeNote(b i18n.WizardStrings, catalog plugincatalog.PluginCatalog, rankedID string) string {
	if rankedID == "" {
		return ""
	}
	provider, ok := plugincatalog.FindCatalogProviderRef(catalog, rankedID)
	if !ok || domain.NormalizeRankedRuntime(provider.Runtime).Kind == domain.RankedRuntimeNone {
		return ""
	}
	return fmt.Sprintf(b.NoteRankedHostNode, rankedID, domain.MinimumOpenSpecNodeVersion)
}

func printRankedRuntimeNote(b i18n.WizardStrings, catalog plugincatalog.PluginCatalog, rankedID string) {
	if note := rankedRuntimeNote(b, catalog, rankedID); note != "" {
		fmt.Println(note)
	}
}

// PromptExecutionSlot asks for the execution slot provider; it falls back to the native execution provider when the
// catalog offers no option for the slot.
func PromptExecutionSlot(p Prompter, b i18n.WizardStrings, catalog plugincatalog.PluginCatalog, providerRisk map[string]string, defaults Defaults) (string, string, error) {
	options := defaults.applyToSlot(plugincatalog.NewInstallOptionSet(catalog).ForSlot(string(domain.SlotExecution)), string(domain.SlotExecution))
	printExcludedCandidates(options.Excluded)
	if len(options.IDs) == 0 {
		// Older synthetic extractors predate the catalogued internal skill.
		provider, err := promptProvider(p, b.PromptExecution, names.NativeExecutionProvider, []string{names.NativeExecutionProvider}, b.LabelCustomInput, providerRisk, names.RiskControlled, string(domain.SlotExecution))
		return provider, domain.SlotBindingModeCustom, err
	}
	return PromptSlotOptions(p, b.PromptExecution, options, b.LabelCustomInput, providerRisk, names.RiskControlled, string(domain.SlotExecution))
}

// printExcludedCandidates prints one line per candidate compatibleProviderOptions
// excluded, naming the candidate and its compatibility reasons, so an
// operator seeing a single-option (native-only) prompt can tell an
// intentional exclusion from a bug.
func printExcludedCandidates(excluded []plugincatalog.ExcludedProviderOption) {
	if err := PrintExcludedCandidatesTo(os.Stdout, excluded); err != nil {
		return
	}
}

// PrintExcludedCandidatesTo writes the candidates excluded from the options, with their reasons, to w.
func PrintExcludedCandidatesTo(w io.Writer, excluded []plugincatalog.ExcludedProviderOption) error {
	for _, candidate := range excluded {
		details := make([]string, 0, len(candidate.Reasons))
		for _, reason := range candidate.Reasons {
			details = append(details, fmt.Sprintf("%s: %s", reason.Code, reason.Detail))
		}
		if _, err := fmt.Fprintf(w, "  %s: excluded — %s\n", candidate.ID, strings.Join(details, "; ")); err != nil {
			return fmt.Errorf("write excluded candidate: %w", err)
		}
	}
	return nil
}
