package leveling

import "strings"

// DisplayName returns the short model name for a provider model id: the
// configured display name, otherwise displayFallback.
func (p Policy) DisplayName(provider, model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return ""
	}
	if profile, ok := p.Providers[strings.ToUpper(strings.TrimSpace(provider))]; ok {
		if name := strings.TrimSpace(profile.Display[model]); name != "" {
			return name
		}
	}
	return displayFallback(provider, model)
}

// displayFallback shortens a model id without reading the policy: it removes a
// leading `<provider>-` and upper-cases the first letter, so `claude-sonnet-5`
// under provider CLAUDE reads `Sonnet-5`. It is the only shortening available
// to a level answered entirely by the manual or host source, because the
// configured display names live in the policy and a complete manual or host
// answer must not load it (see the leveling-wizard-mode on-demand rule).
func displayFallback(provider, model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return ""
	}
	if provider = strings.TrimSpace(provider); provider != "" {
		prefix := strings.ToLower(provider) + "-"
		if len(model) > len(prefix) && strings.EqualFold(model[:len(prefix)], prefix) {
			model = model[len(prefix):]
		}
	}
	return capitalize(model)
}
