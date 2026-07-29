package compile

// IDE-level workspace registration subtests for ideAwareness (.agents/skills.json
// + .agents/AGENTS.md), covering Antigravity's Workspace Customizations Root
// registration mechanism.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIDEAwareness(t *testing.T) {
	t.Parallel()

	t.Run("no-op when .strategist is absent", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		require.NoError(t, ideAwareness(dir))

		_, err := os.Stat(filepath.Join(dir, ".agents"))
		assert.True(t, os.IsNotExist(err), ".agents/ must not be created without .strategist/")
	})

	t.Run("creates skills.json and AGENTS.md when .strategist is present", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".strategist"), 0o755))

		require.NoError(t, ideAwareness(dir))

		skillsRaw, err := os.ReadFile(filepath.Join(dir, ".agents", "skills.json"))
		require.NoError(t, err)
		var doc map[string]any
		require.NoError(t, json.Unmarshal(skillsRaw, &doc))
		entries, ok := doc["entries"].([]any)
		require.True(t, ok)
		require.Len(t, entries, 1)
		assert.Equal(t, ".strategist", entries[0].(map[string]any)["path"])

		agentsMD, err := os.ReadFile(filepath.Join(dir, ".agents", "AGENTS.md"))
		require.NoError(t, err)
		s := string(agentsMD)
		assert.Contains(t, s, agentsRuntimeStartMarker)
		assert.Contains(t, s, agentsRuntimeEndMarker)
		assert.Contains(t, s, "## Strategist Runtime Discovery")
		assert.Contains(t, s, "strategist check")
	})

	t.Run("merges into existing skills.json, preserving other entries", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".strategist"), 0o755))
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".agents"), 0o755))

		initial := `{"entries": [{"path": "some-other-skill"}], "unrelated_key": true}`
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".agents", "skills.json"), []byte(initial), 0o644))

		require.NoError(t, ideAwareness(dir))

		raw, err := os.ReadFile(filepath.Join(dir, ".agents", "skills.json"))
		require.NoError(t, err)
		var doc map[string]any
		require.NoError(t, json.Unmarshal(raw, &doc))
		assert.Equal(t, true, doc["unrelated_key"])
		entries, ok := doc["entries"].([]any)
		require.True(t, ok)
		require.Len(t, entries, 2)
		paths := []any{entries[0].(map[string]any)["path"], entries[1].(map[string]any)["path"]}
		assert.Contains(t, paths, "some-other-skill")
		assert.Contains(t, paths, ".strategist")
	})

	t.Run("does not duplicate .strategist entry on second run", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".strategist"), 0o755))

		require.NoError(t, ideAwareness(dir))
		require.NoError(t, ideAwareness(dir))

		raw, err := os.ReadFile(filepath.Join(dir, ".agents", "skills.json"))
		require.NoError(t, err)
		var doc map[string]any
		require.NoError(t, json.Unmarshal(raw, &doc))
		entries := doc["entries"].([]any)
		assert.Len(t, entries, 1)
	})

	t.Run("preserves unrelated content in existing AGENTS.md, replacing only the delimited block", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".strategist"), 0o755))
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".agents"), 0o755))

		initial := "# Agent Rules\n\n## Other Rules\n\nKeep this.\n\n" +
			agentsRuntimeStartMarker + "\nOld content here.\n" + agentsRuntimeEndMarker + "\n"
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".agents", "AGENTS.md"), []byte(initial), 0o644))

		require.NoError(t, ideAwareness(dir))

		content, err := os.ReadFile(filepath.Join(dir, ".agents", "AGENTS.md"))
		require.NoError(t, err)
		s := string(content)
		assert.Contains(t, s, "## Other Rules")
		assert.Contains(t, s, "Keep this.")
		assert.NotContains(t, s, "Old content here.")
		assert.Contains(t, s, "## Strategist Runtime Discovery")
	})

	t.Run("idempotent on second run", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".strategist"), 0o755))

		require.NoError(t, ideAwareness(dir))
		firstSkills, _ := os.ReadFile(filepath.Join(dir, ".agents", "skills.json"))
		firstAgents, _ := os.ReadFile(filepath.Join(dir, ".agents", "AGENTS.md"))

		require.NoError(t, ideAwareness(dir))
		secondSkills, _ := os.ReadFile(filepath.Join(dir, ".agents", "skills.json"))
		secondAgents, _ := os.ReadFile(filepath.Join(dir, ".agents", "AGENTS.md"))

		assert.Equal(t, string(firstSkills), string(secondSkills))
		assert.Equal(t, string(firstAgents), string(secondAgents))
	})
}
