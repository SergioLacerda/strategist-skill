package install

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/i18n"
)

var langOptions = []string{"en", "pt-BR"}

var installableDefaultProviders = map[string]string{
	"brainstorming":    "skills/brainstorming/skill.yaml",
	"openspec-explore": "skills/openspec-explore/skill.yaml",
}

// knownProviderRisk maps provider ids to their declared risk_score.
// Kept in sync with strategist/templates/known-providers.yaml.
var knownProviderRisk = map[string]string{
	"brainstorming":           "write_analysis",
	"openspec-explore":        "write_analysis",
	"openspec-propose":        "write_analysis",
	"openspec-apply-change":   "controlled",
	"openspec-archive-change": "write_analysis",
	"sdd-ask":                 "controlled",
	"sdd-ask-full":            "controlled",
	"sdd-diagnose":            "write_analysis",
	"sdd-converge":            "controlled",
	"sdd-correct":             "controlled",
	"sdd-stabilize":           "controlled",
	"sdd-validate-governance": "write_analysis",
	"sdd-organize":            "write_analysis",
	"sdd-review-architecture": "write_analysis",
	"archivist":               "write_analysis",
}

// validateProvider returns a non-empty warning if provider is unknown or its
// declared risk_score does not match the expected risk for the slot.
func validateProvider(provider, expectedRisk string) string {
	risk, ok := knownProviderRisk[provider]
	if !ok {
		return fmt.Sprintf("⚠  provider %q is not in the known-providers registry — "+
			"ensure its skill.yaml declares risk_score: %s", provider, expectedRisk)
	}
	if risk != expectedRisk {
		return fmt.Sprintf("⚠  provider %q has risk_score %q but slot requires %q — "+
			"preflight will block at runtime", provider, risk, expectedRisk)
	}
	return ""
}

// runWizard collects install configuration through p.
func runWizard(p Prompter) (domain.WizardConfig, error) {
	// Prompt 1 — bilingual, bundle not yet chosen
	uiLang, err := p.Select("Preferred language / Idioma preferido", "en", langOptions)
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: ui_language: %w", err)
	}
	uiLang = normLang(uiLang)

	b := i18n.BundleFor(uiLang)

	docLang, err := p.Select(b.PromptDocLang, "en", langOptions)
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: doc_language: %w", err)
	}
	chatLang, err := p.Select(b.PromptChatLang, "en", langOptions)
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: chat_language: %w", err)
	}
	codeLang, err := p.Select(b.PromptCodeLang, "en", langOptions)
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: code_language: %w", err)
	}

	mode, err := p.Select(b.PromptMode, "epic", []string{"pragmatic", "epic"})
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: mode: %w", err)
	}
	basePath, err := p.Input(b.PromptBasePath, ".analysis")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: base_path: %w", err)
	}
	adrRaw, err := p.Select(b.PromptAdr, "yes", []string{"yes", "no"})
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: adr_enabled: %w", err)
	}

	executionMode, gitPersistenceMode, err := promptPolicyAndGit(p, b)
	if err != nil {
		return domain.WizardConfig{}, err
	}

	discovery, refinement, execution, err := promptSlots(p, b)
	if err != nil {
		return domain.WizardConfig{}, err
	}

	fmt.Println(b.HeaderChest)
	chestPath, err := p.Input(b.PromptChestPath, "")
	if err != nil {
		return domain.WizardConfig{}, fmt.Errorf("wizard: treasure_chest: %w", err)
	}

	return domain.WizardConfig{
		Mode:               mode,
		BasePath:           basePath,
		ExecutionMode:      executionMode,
		GitPersistenceMode: gitPersistenceMode,
		UILanguage:         uiLang,
		DocLanguage:        normLang(docLang),
		ChatLanguage:       normLang(chatLang),
		CodeLanguage:       normLang(codeLang),
		AdrEnabled:         adrRaw == "yes",
		DiscoveryProvider:  discovery,
		RefinementProvider: refinement,
		ExecutionProvider:  execution,
		TreasureChestPath:  chestPath,
	}, nil
}

// promptPolicyAndGit collects execution mode and git persistence mode, validates the policy.
func promptPolicyAndGit(p Prompter, b i18n.WizardStrings) (executionMode, gitPersistenceMode string, err error) {
	executionMode, err = p.Select(b.PromptExecutionMode, "plan_only", []string{"plan_only", "apply_workspace"})
	if err != nil {
		return "", "", fmt.Errorf("wizard: execution_mode: %w", err)
	}
	gitOptions := []string{"forbidden"}
	if executionMode == domain.ExecutionModeApplyWorkspace {
		gitOptions = []string{"forbidden", "explicit_commit"}
	}
	gitPersistenceMode, err = p.Select(b.PromptGitMode, "forbidden", gitOptions)
	if err != nil {
		return "", "", fmt.Errorf("wizard: git_persistence_mode: %w", err)
	}
	if err = domain.NewMissionPolicy(executionMode, gitPersistenceMode).Validate(); err != nil {
		return "", "", fmt.Errorf("wizard: policy: %w", err)
	}
	return executionMode, gitPersistenceMode, nil
}

// promptSlots collects discovery, refinement and execution slot providers.
func promptSlots(p Prompter, b i18n.WizardStrings) (discovery, refinement, execution string, err error) {
	fmt.Println(b.HeaderSlots)
	discovery, err = p.SelectOrInput(b.PromptDiscovery, "brainstorming", []string{"brainstorming"}, b.LabelCustomInput)
	if err != nil {
		return "", "", "", fmt.Errorf("wizard: discovery: %w", err)
	}
	if w := validateProvider(discovery, "write_analysis"); w != "" {
		fmt.Println(w)
	}
	refinement, err = p.SelectOrInput(b.PromptRefinement, "openspec-explore", []string{"openspec-explore"}, b.LabelCustomInput)
	if err != nil {
		return "", "", "", fmt.Errorf("wizard: refinement: %w", err)
	}
	if w := validateProvider(refinement, "write_analysis"); w != "" {
		fmt.Println(w)
	}
	execution, err = p.SelectOrInput(b.PromptExecution, "sdd-ask", []string{"sdd-ask", "sdd-ask-full"}, b.LabelCustomInput)
	if err != nil {
		return "", "", "", fmt.Errorf("wizard: execution: %w", err)
	}
	if w := validateProvider(execution, "controlled"); w != "" {
		fmt.Println(w)
	}
	return discovery, refinement, execution, nil
}

// normLang normalises language input to canonical form: "en" or "pt-BR".
func normLang(raw string) string {
	if strings.EqualFold(raw, "pt-BR") {
		return "pt-BR"
	}
	return raw
}
