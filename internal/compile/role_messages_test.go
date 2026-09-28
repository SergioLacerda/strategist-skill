package compile

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoleEventAliasesMapOldKeysToTheGenericTemplates(t *testing.T) {
	aliases := RoleEventAliases(domain.DefaultRoleRegistry())
	assert.Equal(t, RoleEventAlias{Generic: "role_start", Role: "ranger"}, aliases["ranger_start"])
	assert.Equal(t, RoleEventAlias{Generic: "role_done", Role: "archivist"}, aliases["archivist_done"])
	assert.Equal(t, RoleEventAlias{Generic: "role_task_done", Role: "sniper"}, aliases["sniper_task_done"])
	assert.NotContains(t, aliases, "scout_start", "pre-pipeline roles have no start/done lines")
	assert.NotContains(t, aliases, "ranger_task_done", "only the execution role has a task line")
}

func TestExpandRoleMessagesUsesDefaultWordingForARoleWithoutPhrases(t *testing.T) {
	reg, err := domain.NewRoleRegistry([]domain.Role{{ID: "scout"}, {ID: "ranger", Slot: "discovery", Phase: 1, Extensibility: domain.RoleExtensibilityPluggable}, {ID: "auditor", Phase: 2}})
	require.NoError(t, err)
	content := map[string]any{
		"role_start": "{role_emoji} {role_title}: {start_text}",
		"role_done":  "{role_title} {done_text} {artifact_label}",
		"role_phrases": map[string]any{
			"_default": map[string]any{"emoji": "E", "start_text": "go", "done_text": "ok", "artifact_label": "At:", "task_text": "must be ignored"},
			"ranger":   map[string]any{"emoji": "R", "start_text": "recon"},
		},
	}
	expandRoleMessages(content, reg)

	assert.Equal(t, "R Ranger: recon", content["ranger_start"], "own wording wins")
	assert.Equal(t, "E Auditor: go", content["auditor_start"], "the default wording covers a new role")
	assert.Equal(t, "Ranger ok At:", content["ranger_done"])
	assert.Equal(t, "Auditor ok At:", content["auditor_done"], "the default wording covers a new role's done line")
	assert.NotContains(t, content, "auditor_task_done", "task wording is opt-in per role")
	for _, key := range []string{"role_start", "role_done", "role_phrases"} {
		assert.NotContains(t, content, key)
	}
}

func TestExpandRoleMessagesLeavesContentWithoutTemplatesUntouched(t *testing.T) {
	content := map[string]any{"intake_summary": "x", "ranger_start": "hand written"}
	expandRoleMessages(content, domain.DefaultRoleRegistry())
	assert.Equal(t, map[string]any{"intake_summary": "x", "ranger_start": "hand written"}, content)
}
