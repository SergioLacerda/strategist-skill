package wizard

import "strings"

// NormLang normalises language input to canonical form: "en" or "pt-BR".
// Accepts "pt" (skill.yaml canonical) and "pt-BR" (legacy/UI form).
func NormLang(raw string) string {
	if strings.EqualFold(raw, "pt-BR") || strings.EqualFold(raw, "pt") {
		return "pt-BR"
	}
	return raw
}
