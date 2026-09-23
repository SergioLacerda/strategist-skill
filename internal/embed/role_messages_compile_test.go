package embed_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// compiledPersonas lays the embedded runtime tree into a temp root, compiles it,
// and returns the compiled personas: the same path `strategist compile` takes.
func compiledPersonas(t *testing.T, mutate func(root string)) map[string]any {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, embed.Extractor{}.Extract(root, false))
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"),
		[]byte("mode: full\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n  refinement: openspec-explore\n  execution: sdd-ask\n"), 0o600))
	if mutate != nil {
		mutate(root)
	}
	out := filepath.Join(root, ".compiled", ".config.gz")
	require.NoError(t, compile.Config(root, out))
	var artifact map[string]any
	testutil.ReadGzJSON(t, out, &artifact)
	personas, ok := artifact["personas"].(map[string]any)
	require.True(t, ok)
	return personas
}

func langMap(t *testing.T, personas map[string]any, persona, section, lang string) map[string]any {
	t.Helper()
	p, ok := personas[persona].(map[string]any)
	require.True(t, ok, persona)
	sec, ok := p[section].(map[string]any)
	require.True(t, ok, "%s.%s", persona, section)
	m, ok := sec[lang].(map[string]any)
	require.True(t, ok, "%s.%s.%s", persona, section, lang)
	return m
}

// The role templates were once hand-written per role, persona and language. The
// generic templates must reproduce those messages exactly: the golden file is the
// pre-generalization output, so any drift is a behavior change.
func TestGenericRoleTemplatesReproduceTheOriginalMessages(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "compile", "testdata", "role_messages_golden.json"))
	require.NoError(t, err)
	var golden map[string]map[string]*string
	require.NoError(t, json.Unmarshal(raw, &golden))

	personas := compiledPersonas(t, nil)
	for name, keys := range golden {
		persona, lang, _ := strings.Cut(name, "/")
		content := langMap(t, personas, persona, "content_by_lang", lang)
		for key, want := range keys {
			if want == nil {
				assert.NotContains(t, content, key, "%s: %s must stay absent", name, key)
				continue
			}
			assert.Equal(t, *want, content[key], "%s: %s", name, key)
		}
	}
}

var (
	genericSourceKeys = []string{"role_start", "role_done", "role_task_done", "role_phrases"}
	// {phase_bar}, {phase_pct} and {phase_mark} are no longer defined by the
	// compile step (the progress line was removed). They stay here as a
	// tripwire: a template that reintroduces them would leak them unexpanded.
	compileTimeTokens = []string{"{role_title}", "{role_emoji}", "{start_text}", "{done_text}", "{artifact_label}", "{task_text}", "{phase_bar}", "{phase_pct}", "{phase_mark}"}
	// progressMarkers identify the removed "channeling mana" progress line.
	progressMarkers = []string{"channeling mana", "█", "▓", "░", "{bar}", "{pct}"}
)

// Role, task and Approval Gate messages convey progress only through the
// mission checkpoint and the phase header; no persona or language renders a
// progress bar or percentage line.
func TestCompiledMessagesCarryNoProgressBar(t *testing.T) {
	personas := compiledPersonas(t, nil)
	for _, persona := range []string{"epic", "pragmatic"} {
		for _, lang := range []string{"en", "pt-BR"} {
			assertNoProgressMarkers(t, persona+"/"+lang, langMap(t, personas, persona, "content_by_lang", lang))
		}
	}
}

func assertNoProgressMarkers(t *testing.T, label string, content map[string]any) {
	t.Helper()
	for key, value := range content {
		text, _ := value.(string)
		for _, marker := range progressMarkers {
			assert.NotContains(t, text, marker, "%s: %s", label, key)
		}
	}
}

func TestCompiledMessagesCarryNoCompileTimePlaceholdersOrGenericKeys(t *testing.T) {
	personas := compiledPersonas(t, nil)
	for _, persona := range []string{"epic", "pragmatic"} {
		for _, lang := range []string{"en", "pt-BR"} {
			assertNoCompileTimeLeaks(t, persona+"/"+lang, langMap(t, personas, persona, "content_by_lang", lang))
		}
	}
}

// assertNoCompileTimeLeaks checks that the generic source keys and the
// compile-time placeholders never reach an agent-facing message.
func assertNoCompileTimeLeaks(t *testing.T, label string, content map[string]any) {
	t.Helper()
	for _, key := range genericSourceKeys {
		assert.NotContains(t, content, key, "%s: generic source key %s must not leak", label, key)
	}
	for key, value := range content {
		assertNoTokens(t, label+": "+key, value)
	}
}

func assertNoTokens(t *testing.T, label string, value any) {
	t.Helper()
	text, _ := value.(string)
	for _, token := range compileTimeTokens {
		assert.NotContains(t, text, token, label)
	}
}

func TestANewRoleGetsGenericMessagesInEveryPersonaAndLanguage(t *testing.T) {
	personas := compiledPersonas(t, func(root string) {
		require.NoError(t, os.WriteFile(filepath.Join(root, "roles", "auditor.yaml"), []byte("role: auditor\nphase: 5\npluggable: false\n"), 0o600))
	})
	for _, persona := range []string{"epic", "pragmatic"} {
		for _, lang := range []string{"en", "pt-BR"} {
			content := langMap(t, personas, persona, "content_by_lang", lang)
			assert.Contains(t, content["auditor_start"], "{role_level_header}", "%s/%s", persona, lang)
			assert.Contains(t, content["auditor_start"], "{mission_id}", "%s/%s", persona, lang)
			assert.Contains(t, content["auditor_done"], "{artifact_path}", "%s/%s", persona, lang)
			assert.NotContains(t, content, "auditor_task_done", "a role without task wording has no task line")
		}
	}
	epic := langMap(t, personas, "epic", "content_by_lang", "en")
	assert.Contains(t, epic["auditor_done"], "Auditor", "the role name is derived from the registry")
	assert.NotContains(t, epic["auditor_done"], "%", "done lines carry no progress percentage")
}

func TestExplicitMessageKeyOverridesTheGeneratedOne(t *testing.T) {
	personas := compiledPersonas(t, func(root string) {
		path := filepath.Join(root, "personas", "epic.yaml")
		raw, err := os.ReadFile(path) //nolint:gosec // temp file
		require.NoError(t, err)
		patched := strings.Replace(string(raw), "    role_start: |", "    ranger_start: \"custom ranger start\"\n    role_start: |", 1)
		require.NotEqual(t, string(raw), patched)
		require.NoError(t, os.WriteFile(path, []byte(patched), 0o600))
	})
	content := langMap(t, personas, "epic", "content_by_lang", "en")
	assert.Equal(t, "custom ranger start", content["ranger_start"], "a hand-written key wins over the generic template")
	assert.Contains(t, content["archivist_start"], "Archivist", "other roles still use the template")
}

func TestRoleEventAliasesResolveInTheCompiledContent(t *testing.T) {
	personas := compiledPersonas(t, nil)
	content := langMap(t, personas, "epic", "content_by_lang", "en")
	for alias := range compile.RoleEventAliases(domain.DefaultRoleRegistry()) {
		assert.Contains(t, content, alias, "alias %s must resolve in the compiled content", alias)
	}
}

func TestListedRoleEventAliasesShareTheGenericLevel(t *testing.T) {
	raw, err := embed.Extractor{}.ReadFile("output-profiles/emit-taxonomy.yaml")
	require.NoError(t, err)
	var taxonomy struct {
		Emits map[string]string `yaml:"emits"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &taxonomy))

	for _, generic := range []string{"role_start", "role_done", "role_task_done"} {
		assert.Equal(t, "INFO", taxonomy.Emits[generic], generic)
	}
	for alias, target := range compile.RoleEventAliases(domain.DefaultRoleRegistry()) {
		level, listed := taxonomy.Emits[alias]
		if !listed {
			continue // an alias absent from the taxonomy keeps today's behavior
		}
		assert.Equal(t, taxonomy.Emits[target.Generic], level, "%s must have the level of %s", alias, target.Generic)
	}
}

// Narration lines use the short inline form: `🎯 **Ranger(Sonnet-High):** ...`.
// The placeholder is empty when the level is unknown, so the line reads as before.
func TestNarrationLinesCarryTheInlineRoleLevelTag(t *testing.T) {
	personas := compiledPersonas(t, nil)
	for _, lang := range []string{"en", "pt-BR"} {
		lines := langMap(t, personas, "epic", "phase_announcements", lang)
		for key, role := range map[string]string{
			"discovery_starting": "Ranger", "discovery_done": "Ranger",
			"refinement_starting": "Archivist", "refinement_done": "Archivist",
			"documentation_starting": "Sniper", "documentation_target_done": "Sniper", "documentation_done": "Sniper",
			"scout_done": "Scout",
		} {
			assert.Contains(t, lines[key], "**"+role+"{role_level_tag}:**", "%s/%s", lang, key)
		}
		assert.NotContains(t, lines["approval_gate_shown"], "{role_level_tag}", "%s: the gate is not a role and has no level", lang)
		assert.Contains(t, lines["scout_done"], "{route}", lang)
	}
}
