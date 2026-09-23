package compile

import (
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/i18n"
)

func injectPTBRRuntime(personasRaw map[string]map[string]any) {
	ptBRRuntime, _ := i18n.RuntimeBundleFor(i18n.LangPTBR)
	ptBRPhaseAnnouncements, _ := i18n.PhaseAnnouncementsFor(i18n.LangPTBR)
	for _, raw := range personasRaw {
		if cbl, ok := contentByLang(raw); ok {
			cbl[i18n.LangPTBR] = ptBRRuntime.ToMap()
		}
		if pa, ok := phaseAnnouncements(raw); ok {
			pa[i18n.LangPTBR] = ptBRPhaseAnnouncements.ToMap()
		}
	}
}

// expandPersonaRoleMessages expands the generic role templates of every language
// of every persona into per-role message keys (see role_messages.go).
func expandPersonaRoleMessages(personasRaw map[string]map[string]any, reg domain.RoleRegistry) {
	for _, raw := range personasRaw {
		if cbl, ok := contentByLang(raw); ok {
			expandLanguageMessages(cbl, reg)
		}
	}
}

func expandLanguageMessages(cbl map[string]any, reg domain.RoleRegistry) {
	for _, langContent := range cbl {
		if content := asMap(langContent); content != nil {
			expandRoleMessages(content, reg)
		}
	}
}

func contentByLang(raw any) (map[string]any, bool) {
	personaMap, ok := raw.(map[string]any)
	if !ok {
		return nil, false
	}
	cbl, ok := personaMap["content_by_lang"].(map[string]any)
	return cbl, ok
}

func phaseAnnouncements(raw any) (map[string]any, bool) {
	personaMap, ok := raw.(map[string]any)
	if !ok {
		return nil, false
	}
	pa, ok := personaMap["phase_announcements"].(map[string]any)
	return pa, ok
}
