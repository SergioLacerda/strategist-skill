package compile_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/i18n"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileConfig_InjectsPTBRRuntimeContentByLang(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"),
		[]byte("mode: full\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "personas", "epic.yaml"), []byte(`id: epic
tone_directive: be precise
phase_labels:
  discovery: analysis
  refinement: refinement
  execution: execution
diagnostics:
  pipeline_header: "[Strategist] pipeline=starting mission_id={id}"
  bootstrap_origin: "[Strategist] profile_path={path} active_yaml={active} reason={reason}"
content_by_lang:
  en:
    intake_summary: "Mission received: {task_type}"
`), 0o644))

	out := filepath.Join(dir, ".compiled", ".config.gz")
	require.NoError(t, compile.Config(dir, out))

	var artifact map[string]any
	testutil.ReadGzJSON(t, out, &artifact)
	personas, ok := artifact["personas"].(map[string]any)
	require.True(t, ok)
	epic, ok := personas["epic"].(map[string]any)
	require.True(t, ok)
	cbl, ok := epic["content_by_lang"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, cbl, i18n.LangEN)

	ptBR, ok := cbl[i18n.LangPTBR].(map[string]any)
	require.True(t, ok, "expected content_by_lang.pt-BR to be injected")
	assert.Equal(t, sortedAnyMapKeys(i18n.PTBRRuntime.ToMap()), sortedAnyMapKeys(ptBR))

	sourcePersona, err := os.ReadFile(filepath.Join(dir, "personas", "epic.yaml"))
	require.NoError(t, err)
	assert.NotContains(t, string(sourcePersona), i18n.LangPTBR)
}

func sortedAnyMapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
