package wizard

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/SergioLacerda/strategist-skill/internal/install/support"
)

// rankedOptionSuffix distinguishes the "Ranger rankeado"-equivalent menu
// entry from its plain Customizado counterpart in the flat option list
// Prompter.SelectOrInput renders (see withRankedOption/splitRankedChoice) —
// docs/adr/0043-ranked-pipeline-pilot-implementation-decisions.md DEC-001's
// "exactly two named choices" model, expressed within the existing Prompter
// contract rather than inventing a second UI abstraction for one pilot.
const rankedOptionSuffix = "::ranked"

const (
	rankedOptionLabel = " (Ranked)"
	customOptionLabel = " (Custom)"
)

// withRankedOption prepends rankedID's synthetic, pre-selected option to
// ids when rankedID is non-empty — DEC-002's "Ranked is pre-selected but
// Customizado's full candidate list is still listed, never replaced." When
// rankedID is empty (no certified Ranked binding for this role), options and
// uiDefault pass through unchanged — task 2.1's own validation: a slot with
// no certified Ranked binding shows only the Customizado options.
func withRankedOption(ids []string, defaultID, rankedID string) (options []string, uiDefault string) {
	if rankedID == "" {
		return ids, defaultID
	}
	rankedOption := rankedID + rankedOptionSuffix
	options = make([]string, 0, len(ids)+1)
	options = append(options, rankedOption)
	options = append(options, ids...)
	return options, rankedOption
}

// splitRankedChoice inverts withRankedOption: choice is the raw string the
// operator selected, and rankedID is the same value withRankedOption was
// called with. Returns the plain provider id plus SlotBindingModeRanked when
// choice is the synthetic Ranked option, or the choice unchanged plus
// SlotBindingModeCustom otherwise — every non-Ranked selection, including a
// free-typed custom skill id, resolves to Custom.
func splitRankedChoice(choice, rankedID string) (provider, mode string) {
	if rankedID != "" && choice == rankedID+rankedOptionSuffix {
		return rankedID, domain.SlotBindingModeRanked
	}
	return choice, domain.SlotBindingModeCustom
}

// WithRankedOptions is withRankedOption for several certified versions: one
// synthetic "<ref>::ranked" entry per certified reference is prepended to the
// Custom list, and the pre-selection is the Ranked default, or the Custom
// default when no version is certified.
func WithRankedOptions(options plugincatalog.SlotOptions) (menu []string, uiDefault string) {
	if len(options.RankedRefs) == 0 {
		return options.IDs, options.DefaultID
	}
	menu = make([]string, 0, len(options.RankedRefs)+len(options.IDs))
	for _, ref := range options.RankedRefs {
		menu = append(menu, ref+rankedOptionSuffix)
	}
	menu = append(menu, options.IDs...)
	return menu, rankedMenuDefault(options)
}

// WithRankedLabeledOptions keeps canonical values separate from the labels
// shown by the interactive UI. A provider that is available in both modes is
// therefore presented as two explicit choices instead of two identical ids.
func WithRankedLabeledOptions(options plugincatalog.SlotOptions) (menu []PromptOption, uiDefault string) {
	if len(options.RankedRefs) == 0 {
		menu = make([]PromptOption, 0, len(options.IDs))
		for _, ref := range options.IDs {
			menu = append(menu, PromptOption{Label: ref, Value: ref})
		}
		return menu, options.DefaultID
	}

	menu = make([]PromptOption, 0, len(options.RankedRefs)+len(options.IDs))
	for _, ref := range options.RankedRefs {
		menu = append(menu, PromptOption{
			Label: ref + rankedOptionLabel,
			Value: ref + rankedOptionSuffix,
		})
	}
	for _, ref := range options.IDs {
		label := ref
		if support.ContainsString(options.RankedRefs, ref) {
			label += customOptionLabel
		}
		menu = append(menu, PromptOption{Label: label, Value: ref})
	}
	return menu, rankedMenuDefault(options)
}

// rankedMenuDefault is the pre-selected menu value when Ranked options exist:
// the Ranked default, or the Custom default when the existing binding is a
// Custom one.
func rankedMenuDefault(options plugincatalog.SlotOptions) string {
	if options.PreferCustom {
		return options.DefaultID
	}
	return options.RankedDefault + rankedOptionSuffix
}

// SplitRankedChoiceAmong inverts withRankedOptions. A "<ref>::ranked" choice is
// Ranked only when ref is one of the offered certified references; every other
// choice, including a free-typed id, is Custom and passes through unchanged.
func SplitRankedChoiceAmong(choice string, rankedRefs []string) (provider, mode string) {
	if ref, found := strings.CutSuffix(choice, rankedOptionSuffix); found {
		for _, offered := range rankedRefs {
			if ref == offered {
				return ref, domain.SlotBindingModeRanked
			}
		}
	}
	return choice, domain.SlotBindingModeCustom
}

// PromptSlotOptions prompts for a slot's provider from versioned options and
// returns the chosen reference plus the pipeline mode it resolved to.
func PromptSlotOptions(p Prompter, prompt string, options plugincatalog.SlotOptions, customLabel string, providerRisk map[string]string, expectedRisk, field string) (provider, mode string, err error) {
	var choice string
	if labeled, ok := p.(labeledPrompter); ok {
		menu, uiDefault := WithRankedLabeledOptions(options)
		choice, err = labeled.selectOrInputWithLabels(prompt, uiDefault, menu, customLabel)
	} else {
		menu, uiDefault := WithRankedOptions(options)
		choice, err = p.SelectOrInput(prompt, uiDefault, menu, customLabel)
	}
	if err != nil {
		return "", "", fmt.Errorf("wizard: %s: %w", field, err)
	}
	provider, mode = SplitRankedChoiceAmong(choice, options.RankedRefs)
	if w := ValidateProvider(providerRisk, DomainProviderID(provider), expectedRisk); w != "" {
		fmt.Println(w)
	}
	return provider, mode, nil
}

// DomainProviderID is the id part of a Weapon reference, for checks keyed by id.
func DomainProviderID(ref string) string {
	id, _ := domain.ParseWeaponRef(ref)
	return id
}
