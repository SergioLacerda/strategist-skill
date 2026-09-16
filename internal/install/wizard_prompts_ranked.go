package install

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// rankedOptionSuffix distinguishes the "Ranger rankeado"-equivalent menu
// entry from its plain Customizado counterpart in the flat option list
// Prompter.SelectOrInput renders (see withRankedOption/splitRankedChoice) —
// docs/adr/0043-ranked-pipeline-pilot-implementation-decisions.md DEC-001's
// "exactly two named choices" model, expressed within the existing Prompter
// contract rather than inventing a second UI abstraction for one pilot.
const rankedOptionSuffix = "::ranked"

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

// promptSlotProvider prompts for a discovery/refinement slot's provider,
// presenting the certified-Ranked option (if any) alongside Custom's full
// candidate list, and returns the resolved plain provider id plus which
// pipeline mode the operator's choice resolved to.
func promptSlotProvider(p Prompter, prompt string, ids []string, defaultID, rankedID, customLabel string, providerRisk map[string]string, expectedRisk, field string) (provider, mode string, err error) {
	options, uiDefault := withRankedOption(ids, defaultID, rankedID)
	choice, err := p.SelectOrInput(prompt, uiDefault, options, customLabel)
	if err != nil {
		return "", "", fmt.Errorf("wizard: %s: %w", field, err)
	}
	provider, mode = splitRankedChoice(choice, rankedID)
	if w := validateProvider(providerRisk, provider, expectedRisk); w != "" {
		fmt.Println(w)
	}
	return provider, mode, nil
}
