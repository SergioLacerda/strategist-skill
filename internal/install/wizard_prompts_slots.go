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
func promptSlots(p Prompter, b i18n.WizardStrings, catalog pluginCatalog, providerRisk map[string]string) (discovery, refinement, execution, discoveryMode, refinementMode, executionMode string, err error) {
	fmt.Println(b.HeaderSlots)

	discoveryIDs, discoveryDefault, discoveryRankedID, discoveryExcluded := compatibleProviderOptions(catalog, "ranger", domain.RoleHandoffSchema["ranger"])
	printExcludedCandidates(discoveryExcluded)
	if len(discoveryIDs) == 0 {
		return "", "", "", "", "", "", fmt.Errorf("wizard: discovery: no compatible weapon for role ranger")
	}
	discovery, discoveryMode, err = promptSlotProvider(p, b.PromptDiscovery, discoveryIDs, discoveryDefault, discoveryRankedID, b.LabelCustomInput, providerRisk, "write_analysis", "discovery")
	if err != nil {
		return "", "", "", "", "", "", err
	}

	refinementIDs, refinementDefault, refinementRankedID, refinementExcluded := compatibleProviderOptions(catalog, "archivist", domain.RoleHandoffSchema["archivist"])
	printExcludedCandidates(refinementExcluded)
	if len(refinementIDs) == 0 {
		return "", "", "", "", "", "", fmt.Errorf("wizard: refinement: no compatible weapon for role archivist")
	}
	refinement, refinementMode, err = promptSlotProvider(p, b.PromptRefinement, refinementIDs, refinementDefault, refinementRankedID, b.LabelCustomInput, providerRisk, "write_analysis", "refinement")
	if err != nil {
		return "", "", "", "", "", "", err
	}
	execution, executionMode, err = promptExecutionSlot(p, b, catalog, providerRisk)
	if err != nil {
		return "", "", "", "", "", "", err
	}
	return discovery, refinement, execution, discoveryMode, refinementMode, executionMode, nil
}

func promptExecutionSlot(p Prompter, b i18n.WizardStrings, catalog pluginCatalog, providerRisk map[string]string) (string, string, error) {
	ids, defaultID, rankedID, excluded := compatibleProviderOptions(catalog, "sniper", "")
	printExcludedCandidates(excluded)
	if len(ids) == 0 {
		// Older synthetic extractors predate the catalogued internal skill.
		provider, err := promptProvider(p, b.PromptExecution, nativeExecutionProvider, []string{nativeExecutionProvider}, b.LabelCustomInput, providerRisk, "controlled", "execution")
		return provider, domain.SlotBindingModeCustom, err
	}
	return promptSlotProvider(p, b.PromptExecution, ids, defaultID, rankedID, b.LabelCustomInput, providerRisk, "controlled", "execution")
}

// excludedProviderOption records why compatibleProviderOptions did not offer
// a given catalog candidate, so promptSlots can print it instead of letting
// the operator wonder whether an empty-looking option list is a bug or an
// intended exclusion (see .analysis/refined/
// 20260914-wizard-weapon-options-not-listed/design.md Task 3).
type excludedProviderOption struct {
	id      string
	reasons []domain.CompatibilityReason
}

// compatibleProviderOptions returns the catalog candidate IDs for roleName
// that role-affinity validation reports compatible, plus
// which one should be pre-selected: whichever compatible candidate is
// marked default in the catalog, or the first compatible one otherwise; the
// id of the certified-Ranked candidate for roleName, if any (empty when
// none — docs/adr/0043-ranked-pipeline-pilot-implementation-decisions.md);
// and the candidates that were excluded along with their compatibility
// reasons. When nothing is compatible, the id list is empty and the caller
// must fail before activation. Native roles are not substitutes for the
// required discovery/refinement weapons.
func compatibleProviderOptions(catalog pluginCatalog, roleName, handoffSchema string) (ids []string, defaultID, rankedID string, excluded []excludedProviderOption) {
	role := domain.RoleContract{
		SchemaVersion: domain.RoleContractSchemaVersion,
		Role:          roleName,
		HandoffSchema: handoffSchema,
	}
	for _, candidate := range providerContractsForRole(catalog, roleName) {
		ids, defaultID, excluded = appendProviderOption(ids, defaultID, excluded, candidate, role)
		if candidate.Ranked && candidate.CertificationDigest != "" {
			rankedID = candidate.ID
		}
	}
	if defaultID == "" && len(ids) > 0 {
		defaultID = ids[0]
	}
	return ids, defaultID, rankedID, excluded
}

func appendProviderOption(ids []string, defaultID string, excluded []excludedProviderOption, candidate domain.ProviderContract, role domain.RoleContract) ([]string, string, []excludedProviderOption) {
	if candidate.Source == domain.ProviderSourceNativeRole && (!candidate.Ranked || candidate.CertificationDigest == "") {
		return ids, defaultID, excluded
	}
	result := candidate.CheckRoleAffinity(role)
	if !result.Compatible {
		return ids, defaultID, append(excluded, excludedProviderOption{id: candidate.ID, reasons: result.Reasons})
	}
	ids = append(ids, candidate.ID)
	if candidate.Default {
		defaultID = candidate.ID
	}
	return ids, defaultID, excluded
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
