package compile

import (
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// Generic role message templates. A persona (or language bundle) declares three
// templates and a phrase table; the compile step expands them once per
// registered role into the concrete keys agents already read (`ranger_start`,
// `archivist_done`, ...), so the old keys stay valid as aliases and a new role
// needs no new template strings.
const (
	roleStartTemplate    = "role_start"
	roleDoneTemplate     = "role_done"
	roleTaskDoneTemplate = "role_task_done"
	rolePhrasesKey       = "role_phrases"
	defaultPhrasesKey    = "_default"
)

// RoleEventAlias names the generic template an old per-role event key expands.
type RoleEventAlias struct {
	Generic string
	Role    string
}

// RoleEventAliases lists the concrete event keys generated from the generic
// templates: `<role>_start` and `<role>_done` for every role with a checkpoint
// phase, and `<role>_task_done` for the role that fills the execution slot (the
// only one that materializes tasks). Pre-pipeline roles have no such lines.
func RoleEventAliases(reg domain.RoleRegistry) map[string]RoleEventAlias {
	aliases := map[string]RoleEventAlias{}
	for _, role := range reg.Roles() {
		if role.Phase <= 0 {
			continue
		}
		aliases[role.ID+"_start"] = RoleEventAlias{Generic: roleStartTemplate, Role: role.ID}
		aliases[role.ID+"_done"] = RoleEventAlias{Generic: roleDoneTemplate, Role: role.ID}
		if role.Slot == string(domain.SlotExecution) {
			aliases[role.ID+"_task_done"] = RoleEventAlias{Generic: roleTaskDoneTemplate, Role: role.ID}
		}
	}
	return aliases
}

// expandRoleMessages expands the generic role templates found in one language's
// content into concrete per-role keys, then removes the generic source keys so
// compile-time placeholders never reach an agent. A key already present in the
// content wins over the generated one, so a hand-written message still works.
func expandRoleMessages(content map[string]any, reg domain.RoleRegistry) {
	phrases := asMap(content[rolePhrasesKey])
	for _, role := range reg.Roles() {
		if role.Phase <= 0 {
			continue
		}
		phrase := rolePhraseFor(phrases, role.ID)
		replacer := strings.NewReplacer(
			"{role_title}", roleDisplayName(role.ID),
			"{role_emoji}", phrase["emoji"],
			"{start_text}", phrase["start_text"],
			"{done_text}", phrase["done_text"],
			"{artifact_label}", phrase["artifact_label"],
			"{task_text}", phrase["task_text"],
		)
		expandOne(content, role.ID+"_start", roleStartTemplate, replacer, true)
		expandOne(content, role.ID+"_done", roleDoneTemplate, replacer, phrase["done_text"] != "")
		expandOne(content, role.ID+"_task_done", roleTaskDoneTemplate, replacer, phrase["task_text"] != "")
	}
	delete(content, roleStartTemplate)
	delete(content, roleDoneTemplate)
	delete(content, roleTaskDoneTemplate)
	delete(content, rolePhrasesKey)
}

func expandOne(content map[string]any, key, templateKey string, replacer *strings.Replacer, enabled bool) {
	if _, exists := content[key]; exists || !enabled {
		return
	}
	if template, ok := content[templateKey].(string); ok && template != "" {
		content[key] = replacer.Replace(template)
	}
}

// rolePhraseFor returns the phrase table of a role, falling back to the
// `_default` entry so a role without its own wording still renders. The default
// supplies emoji and generic wording only: task wording is opt-in per role.
func rolePhraseFor(phrases map[string]any, id string) map[string]string {
	out := stringFields(asMap(phrases[defaultPhrasesKey]))
	delete(out, "task_text")
	for field, value := range stringFields(asMap(phrases[id])) {
		out[field] = value
	}
	return out
}

// stringFields keeps the string-valued entries of a phrase table.
func stringFields(entry map[string]any) map[string]string {
	out := make(map[string]string, len(entry))
	for field, value := range entry {
		if text, ok := value.(string); ok {
			out[field] = text
		}
	}
	return out
}

func roleDisplayName(id string) string {
	if id == "" {
		return ""
	}
	return strings.ToUpper(id[:1]) + id[1:]
}

// asMap returns v as a map, or nil when it is not one (a missing phrase table
// simply yields no wording).
func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}
