package install

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/i18n"
)

// promptSlots collects discovery and refinement slot providers. The execution slot is
// always the native `sniper` role — Strategist's built-in execution persona, not a
// governance/provider skill selectable from `.sdd/skills` (see mission
// 2026-07-25-wizard-execution-slot-native-sniper). The legacy execution prompt is still
// shown and consumed here so prompt count and scripted-input ordering stay stable, but
// its returned value is discarded: no typed input (e.g. `sdd-ask`) can ever leak into
// slots.execution.
//
// The discovery/refinement option lists are driven by explicit role affinity
// (compatibleProviderOptions), not a hardcoded slice. Handoff production is a
// fixed role checkpoint, so a weapon is not hidden merely because its adapter
// does not declare the handoff schema.
func promptSlots(p Prompter, b i18n.WizardStrings, catalog pluginCatalog, providerRisk map[string]string) (discovery, refinement, execution, discoveryMode, refinementMode string, err error) {
	fmt.Println(b.HeaderSlots)

	discoveryIDs, discoveryDefault, discoveryRankedID, discoveryExcluded := compatibleProviderOptions(catalog, "ranger", domain.RoleHandoffSchema["ranger"])
	printExcludedCandidates(discoveryExcluded)
	if len(discoveryIDs) == 0 {
		return "", "", "", "", "", fmt.Errorf("wizard: discovery: no compatible weapon for role ranger")
	}
	discovery, discoveryMode, err = promptSlotProvider(p, b.PromptDiscovery, discoveryIDs, discoveryDefault, discoveryRankedID, b.LabelCustomInput, providerRisk, "write_analysis", "discovery")
	if err != nil {
		return "", "", "", "", "", err
	}

	refinementIDs, refinementDefault, refinementRankedID, refinementExcluded := compatibleProviderOptions(catalog, "archivist", domain.RoleHandoffSchema["archivist"])
	printExcludedCandidates(refinementExcluded)
	if len(refinementIDs) == 0 {
		return "", "", "", "", "", fmt.Errorf("wizard: refinement: no compatible weapon for role archivist")
	}
	refinement, refinementMode, err = promptSlotProvider(p, b.PromptRefinement, refinementIDs, refinementDefault, refinementRankedID, b.LabelCustomInput, providerRisk, "write_analysis", "refinement")
	if err != nil {
		return "", "", "", "", "", err
	}
	if _, err = promptProvider(p, b.PromptExecution, nativeExecutionProvider, []string{nativeExecutionProvider}, b.LabelCustomInput, providerRisk, "controlled", "execution"); err != nil {
		return "", "", "", "", "", err
	}
	return discovery, refinement, nativeExecutionProvider, discoveryMode, refinementMode, nil
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
	if candidate.Source == domain.ProviderSourceNativeRole {
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
	for _, candidate := range excluded {
		details := make([]string, 0, len(candidate.reasons))
		for _, reason := range candidate.reasons {
			details = append(details, fmt.Sprintf("%s: %s", reason.Code, reason.Detail))
		}
		fmt.Printf("  %s: excluded — %s\n", candidate.id, strings.Join(details, "; "))
	}
}
