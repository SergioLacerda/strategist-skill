package embed_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type personaDoc struct {
	ContentByLang map[string]map[string]any `yaml:"content_by_lang"`
	JSONLSchema   struct {
		Fields []string `yaml:"fields"`
	} `yaml:"jsonl_event_schema"`
}

func loadPersona(t *testing.T, id string) personaDoc {
	t.Helper()
	raw, err := embed.Extractor{}.ReadFile("personas/" + id + ".yaml")
	require.NoError(t, err)
	var doc personaDoc
	require.NoError(t, yaml.Unmarshal(raw, &doc))
	return doc
}

// The per-role lines are generated from generic templates at compile time (see
// internal/compile), so the header is asserted on the templates and the gate.
func TestRenderedPersonasCarryRoleLevelHeaderOnGenericRoleTemplates(t *testing.T) {
	for _, id := range []string{"epic", "pragmatic"} {
		en := loadPersona(t, id).ContentByLang["en"]
		header, ok := en["role_level_header"].(string)
		require.True(t, ok, "%s: role_level_header missing", id)
		assert.Contains(t, header, "Fase: {phase_index}/04")
		assert.Contains(t, header, "{model_effort}")
		for _, key := range []string{"role_start", "role_done", "approval_gate_prompt"} {
			assert.Contains(t, en[key], "{role_level_header}", "%s: %s", id, key)
		}
	}
	assert.Contains(t, loadPersona(t, "epic").ContentByLang["en"]["role_task_done"], "{role_level_header}")
}

func TestDebugPersonaJSONLSchemaCarriesLevelFields(t *testing.T) {
	fields := loadPersona(t, "debug").JSONLSchema.Fields
	for _, want := range []string{"role", "model", "effort", "level_source"} {
		assert.Contains(t, fields, want)
	}
}
