package leveling

import "strings"

// DisplayName returns the short model name for a provider model id: the
// configured display name, otherwise the id without its `<provider>-` prefix
// and with its first letter upper-cased.
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
	prefix := strings.ToLower(strings.TrimSpace(provider)) + "-"
	if len(model) > len(prefix) && strings.EqualFold(model[:len(prefix)], prefix) {
		model = model[len(prefix):]
	}
	return capitalize(model)
}
