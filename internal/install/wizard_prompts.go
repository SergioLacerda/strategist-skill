package install

import (
	"fmt"

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
// The discovery/refinement option lists are compatibility-driven
// (compatibleProviderOptions), not a hardcoded slice: a weapon whose
// declared supported_handoff_schemas doesn't match its role's
// HandoffSchema simply stops being offered, and the native role becomes
// the shown default. This is what keeps discovery effectively locked to
// Ranger without a special case — none of discovery's external weapons
// declare a matching schema (see external-skills-source/brainstorming and
// openspec-explore's strategist.yaml), so compatibleProviderOptions always
// resolves to the single native ranger option there today.
func promptSlots(p Prompter, b i18n.WizardStrings, catalog pluginCatalog, providerRisk map[string]string) (discovery, refinement, execution string, err error) {
	fmt.Println(b.HeaderSlots)

	discoveryIDs, discoveryDefault := compatibleProviderOptions(catalog, "ranger", roleHandoffSchema["ranger"], "ranger")
	discovery, err = promptProvider(p, b.PromptDiscovery, discoveryDefault, discoveryIDs, b.LabelCustomInput, providerRisk, "write_analysis", "discovery")
	if err != nil {
		return "", "", "", err
	}

	refinementIDs, refinementDefault := compatibleProviderOptions(catalog, "archivist", roleHandoffSchema["archivist"], "archivist")
	refinement, err = promptProvider(p, b.PromptRefinement, refinementDefault, refinementIDs, b.LabelCustomInput, providerRisk, "write_analysis", "refinement")
	if err != nil {
		return "", "", "", err
	}
	if _, err = promptProvider(p, b.PromptExecution, nativeExecutionProvider, []string{nativeExecutionProvider}, b.LabelCustomInput, providerRisk, "controlled", "execution"); err != nil {
		return "", "", "", err
	}
	return discovery, refinement, nativeExecutionProvider, nil
}

// compatibleProviderOptions returns the catalog candidate IDs for roleName
// that CheckRoleCompatibility reports compatible against handoffSchema, plus
// which one should be pre-selected: whichever compatible candidate is
// marked default in the catalog, or the first compatible one otherwise.
// When nothing is compatible (e.g. every external weapon for this role is
// honestly declared unable to produce the role's handoff shape), it falls
// back to a single-item list naming the catalog's native_role candidate for
// roleName, or fallbackID if the catalog has none — either way the wizard
// stays usable and the operator is not offered a weapon known not to work.
func compatibleProviderOptions(catalog pluginCatalog, roleName, handoffSchema, fallbackID string) (ids []string, defaultID string) {
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
		if !candidate.CheckRoleCompatibility(role).Compatible {
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
	return ids, defaultID
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
