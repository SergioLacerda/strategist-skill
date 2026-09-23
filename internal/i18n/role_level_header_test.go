package i18n_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every role-attributed line must carry the role level header in every language,
// or the model/effort feedback silently disappears for that language.
func TestRoleLinesCarryRoleLevelHeaderInEveryLanguage(t *testing.T) {
	roleKeys := []string{"role_start", "role_done", "role_task_done", "approval_gate_prompt"}
	for name, bundle := range map[string]i18n.RuntimeMessages{"en": i18n.ENRuntime, "pt-BR": i18n.PTBRRuntime} {
		m := bundle.ToMap()
		header, ok := m["role_level_header"].(string)
		require.True(t, ok, "%s: role_level_header key missing", name)
		for _, token := range []string{"{phase_index}", "{role_name}", "{model_effort}"} {
			assert.Contains(t, header, token, "%s header", name)
		}
		for _, key := range roleKeys {
			assert.Contains(t, m[key], "{role_level_header}", "%s: %s", name, key)
		}
	}
}

func TestRoleLevelHeaderIsIdenticalAcrossLanguages(t *testing.T) {
	assert.Equal(t, i18n.ENRuntime.RoleLevelHeader, i18n.PTBRRuntime.RoleLevelHeader, "the Fase/role/Model-Effort layout is fixed, not translated")
}
