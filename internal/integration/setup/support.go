package setup

import (
	"errors"
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/integration/config"
	"github.com/SergioLacerda/strategist-skill/internal/integration/credential"
	"github.com/SergioLacerda/strategist-skill/internal/integration/handoffconsumer"
)

// DataCategories lists the contract fields a handoff pre-check may send, the
// categories shown to the operator at consent time.
func DataCategories() []string {
	var categories []string
	seen := map[string]bool{}
	for _, transition := range []string{handoff.TransitionRangerToArchivist, handoff.TransitionArchivistToSniper} {
		for _, field := range handoffconsumer.Fields(transition) {
			if !seen[field] {
				seen[field] = true
				categories = append(categories, field)
			}
		}
	}
	return categories
}

// DefaultChoice is the preselected answer: keep an existing decision, otherwise
// yes only when the configured credential resolves, no when it does not.
func DefaultChoice(credentialResolves, found bool) Choice {
	switch {
	case found:
		return ChoiceKeep
	case credentialResolves:
		return ChoiceEnable
	default:
		return ChoiceDisable
	}
}

// Offered lists the selections a prompt shows; keep needs a prior decision.
func Offered(found bool) []Choice {
	if found {
		return []Choice{ChoiceEnable, ChoiceDisable, ChoiceKeep}
	}
	return []Choice{ChoiceEnable, ChoiceDisable}
}

// Apply persists the plan atomically when it changes the decision.
func Apply(path string, plan Plan) error {
	if !plan.Write {
		return nil
	}
	if err := config.Save(path, plan.File); err != nil {
		return fmt.Errorf("persist integration decision: %w", err)
	}
	return nil
}

// Resolves reports whether a credential reference resolves for a workspace.
func Resolves(ref, workspace string) bool {
	_, err := credential.Resolve(ref, credential.WorkspaceEnv(workspace))
	return err == nil
}

var errNoDecision = errors.New("no integration decision recorded")

// Decision returns the provider the operator recorded, or an error when none was.
func Decision(file config.File, found bool) (config.Provider, error) {
	provider, ok := file.Providers[ProviderName]
	if !found || !ok {
		return config.Provider{}, errNoDecision
	}
	return provider, nil
}
