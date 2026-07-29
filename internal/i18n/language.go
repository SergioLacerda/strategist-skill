package i18n

import "strings"

const (
	// LangEN is the canonical English language code.
	LangEN = "en"
	// LangPTBR is the canonical Brazilian Portuguese language code.
	LangPTBR = "pt-BR"
)

var (
	wizardBundlesByLang = map[string]WizardStrings{
		LangEN:   EN,
		LangPTBR: PT,
	}
	runtimeBundlesByLang = map[string]RuntimeMessages{
		LangEN:   ENRuntime,
		LangPTBR: PTBRRuntime,
	}
	phaseAnnouncementsByLang = map[string]PhaseAnnouncementsMessages{
		LangPTBR: PTBRPhaseAnnouncements,
	}
)

// NormalizeLang canonicalizes supported language codes used by Strategist.
func NormalizeLang(lang string) string {
	normalized := strings.TrimSpace(lang)
	switch strings.ToLower(normalized) {
	case "", "en", "en-us", "en_us":
		return LangEN
	case "pt", "pt-br", "pt_br":
		return LangPTBR
	default:
		return normalized
	}
}

// SupportedRuntimeLanguages returns runtime bundle language codes in stable order.
func SupportedRuntimeLanguages() []string {
	return []string{LangEN, LangPTBR}
}

// RuntimeBundleFor returns the runtime message bundle for a supported language.
func RuntimeBundleFor(lang string) (RuntimeMessages, bool) {
	bundle, ok := runtimeBundlesByLang[NormalizeLang(lang)]
	return bundle, ok
}

// PhaseAnnouncementsFor returns phase_announcements for languages that define them.
func PhaseAnnouncementsFor(lang string) (PhaseAnnouncementsMessages, bool) {
	bundle, ok := phaseAnnouncementsByLang[NormalizeLang(lang)]
	return bundle, ok
}

func wizardBundleFor(lang string) (WizardStrings, bool) {
	bundle, ok := wizardBundlesByLang[NormalizeLang(lang)]
	return bundle, ok
}
