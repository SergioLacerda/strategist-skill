package install

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/i18n"
)

func promptLanguages(p Prompter, skillCfg skillConfig) (uiLang, docLang, chatLang, codeLang string, b i18n.WizardStrings, err error) {
	uiLang, err = p.Select("Preferred language / Idioma preferido", "en", skillCfg.LangOptions)
	if err != nil {
		err = fmt.Errorf("wizard: ui_language: %w", err)
		return
	}
	uiLang = normLang(uiLang)
	b = i18n.BundleFor(uiLang)
	docLang, err = selectLang(p, b.PromptDocLang, skillCfg.LangOptions, "doc_language")
	if err != nil {
		return
	}
	chatLang, err = selectLang(p, b.PromptChatLang, skillCfg.LangOptions, "chat_language")
	if err != nil {
		return
	}
	codeLang, err = selectLang(p, b.PromptCodeLang, skillCfg.LangOptions, "code_language")
	return
}

func selectLang(p Prompter, prompt string, options []string, field string) (string, error) {
	value, err := p.Select(prompt, "en", options)
	if err != nil {
		return "", fmt.Errorf("wizard: %s: %w", field, err)
	}
	return value, nil
}

func promptWorkspace(p Prompter, b i18n.WizardStrings, skillCfg skillConfig) (string, string, error) {
	mode, err := p.Select(b.PromptMode, "epic", skillCfg.ModeOptions)
	if err != nil {
		return "", "", fmt.Errorf("wizard: mode: %w", err)
	}
	basePath, err := p.Input(b.PromptBasePath, ".analysis")
	if err != nil {
		return "", "", fmt.Errorf("wizard: base_path: %w", err)
	}
	return mode, basePath, nil
}

func promptTreasureChest(p Prompter, b i18n.WizardStrings) (string, error) {
	fmt.Println(b.HeaderChest)
	chestPath, err := p.Input(b.PromptChestPath, "")
	if err != nil {
		return "", fmt.Errorf("wizard: treasure_chest: %w", err)
	}
	if chestPath == "" {
		fmt.Println(b.SkipChestHint)
	}
	return chestPath, nil
}

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
func promptSlots(p Prompter, b i18n.WizardStrings, catalog pluginCatalog, providerRisk map[string]string) (discovery, refinement, execution string, err error) {
	fmt.Println(b.HeaderSlots)

	discoveryIDs, discoveryDefault, discoveryExcluded := compatibleProviderOptions(catalog, "ranger", domain.RoleHandoffSchema["ranger"], "ranger")
	printExcludedCandidates(discoveryExcluded)
	discovery, err = promptProvider(p, b.PromptDiscovery, discoveryDefault, discoveryIDs, b.LabelCustomInput, providerRisk, "write_analysis", "discovery")
	if err != nil {
		return "", "", "", err
	}

	refinementIDs, refinementDefault, refinementExcluded := compatibleProviderOptions(catalog, "archivist", domain.RoleHandoffSchema["archivist"], "archivist")
	printExcludedCandidates(refinementExcluded)
	refinement, err = promptProvider(p, b.PromptRefinement, refinementDefault, refinementIDs, b.LabelCustomInput, providerRisk, "write_analysis", "refinement")
	if err != nil {
		return "", "", "", err
	}
	if _, err = promptProvider(p, b.PromptExecution, nativeExecutionProvider, []string{nativeExecutionProvider}, b.LabelCustomInput, providerRisk, "controlled", "execution"); err != nil {
		return "", "", "", err
	}
	return discovery, refinement, nativeExecutionProvider, nil
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
// marked default in the catalog, or the first compatible one otherwise, and
// the candidates that were excluded along with their compatibility reasons.
// When nothing is compatible (e.g. every external weapon for this role is
// honestly declared unable to produce the role's handoff shape), the id list
// falls back to a single-item list naming the catalog's native_role
// candidate for roleName, or fallbackID if the catalog has none — either way
// the wizard stays usable and the operator is not offered a weapon known not
// to work; the native role itself is never reported as excluded.
func compatibleProviderOptions(catalog pluginCatalog, roleName, handoffSchema, fallbackID string) (ids []string, defaultID string, excluded []excludedProviderOption) {
	role := domain.RoleContract{
		SchemaVersion: domain.RoleContractSchemaVersion,
		Role:          roleName,
		HandoffSchema: handoffSchema,
	}
	var nativeID string
	for _, candidate := range providerContractsForRole(catalog, roleName) {
		if candidate.Source == domain.ProviderSourceNativeRole {
			nativeID = candidate.ID
		}
		result := candidate.CheckRoleAffinity(role)
		if !result.Compatible {
			if candidate.Source != domain.ProviderSourceNativeRole {
				excluded = append(excluded, excludedProviderOption{id: candidate.ID, reasons: result.Reasons})
			}
			continue
		}
		ids = append(ids, candidate.ID)
		if candidate.Default {
			defaultID = candidate.ID
		}
	}
	if defaultID == "" && len(ids) > 0 {
		defaultID = ids[0]
	}
	if len(ids) == 0 {
		if nativeID == "" {
			nativeID = fallbackID
		}
		ids = []string{nativeID}
		defaultID = nativeID
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

func promptProvider(p Prompter, prompt, defaultVal string, options []string, customLabel string, providerRisk map[string]string, expectedRisk, field string) (string, error) {
	provider, err := p.SelectOrInput(prompt, defaultVal, options, customLabel)
	if err != nil {
		return "", fmt.Errorf("wizard: %s: %w", field, err)
	}
	if w := validateProvider(providerRisk, provider, expectedRisk); w != "" {
		fmt.Println(w)
	}
	return provider, nil
}
