package compile

// JSON codex-seed upsert subtests, split from the markdown-section subtests
// in agent_awareness_test.go. Moved to package compile alongside the
// AgentAwareness -> agentAwareness unexport.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentAwareness_CodexSeed(t *testing.T) {
	t.Parallel()

	t.Run("upserts codex required_context when file exists", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		codexPath := filepath.Join(dir, ".sdd", "seedlings", "codex.seed.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(codexPath), 0o755))
		initial := map[string]any{
			"auto_activate":    true,
			"required_context": []any{".sdd/metadata.json", ".strategist/SKILL.md"},
		}
		data, _ := json.Marshal(initial)
		require.NoError(t, os.WriteFile(codexPath, data, 0o644))

		require.NoError(t, agentAwareness(dir))

		var result map[string]any
		raw, _ := os.ReadFile(codexPath)
		require.NoError(t, json.Unmarshal(raw, &result))

		ctx := result["required_context"].([]any)
		assert.Equal(t, ".strategist/agent-protocol.md", ctx[0], "agent-protocol.md must be first")
		assert.Contains(t, ctx, ".sdd/metadata.json")
		assert.Contains(t, ctx, ".strategist/SKILL.md")
		assert.NotNil(t, result["on_strategist_invoke"])
	})

	t.Run("codex seed carries the Strategist Active role-lock header and forbidden actions", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		codexPath := filepath.Join(dir, ".sdd", "seedlings", "codex.seed.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(codexPath), 0o755))
		initial := map[string]any{"required_context": []any{}}
		data, _ := json.Marshal(initial)
		require.NoError(t, os.WriteFile(codexPath, data, 0o644))

		require.NoError(t, agentAwareness(dir))

		var result map[string]any
		raw, _ := os.ReadFile(codexPath)
		require.NoError(t, json.Unmarshal(raw, &result))

		invoke := result["on_strategist_invoke"].(map[string]any)
		assert.Equal(t, "Strategist Active", invoke["header"])
		assert.Contains(t, invoke["role_lock"], "do not solve the task directly")
		assert.Contains(t, invoke["allowed_actions"], "invoke_providers")
		assert.Contains(t, invoke["forbidden_actions"], "direct_execution")
		assert.Contains(t, invoke["forbidden_actions"], "code_or_test_mutation")
		assert.Contains(t, invoke["forbidden_actions"], "provider_fallback")
		assert.Equal(t, "emit error=role_invocation_failed with slot and provider, then stop", invoke["on_role_invocation_failed"])
	})

	t.Run("does not duplicate agent-protocol.md in codex required_context", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		codexPath := filepath.Join(dir, ".sdd", "seedlings", "codex.seed.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(codexPath), 0o755))
		initial := map[string]any{
			"required_context": []any{".strategist/agent-protocol.md", ".sdd/metadata.json"},
		}
		data, _ := json.Marshal(initial)
		require.NoError(t, os.WriteFile(codexPath, data, 0o644))

		require.NoError(t, agentAwareness(dir))

		var result map[string]any
		raw, _ := os.ReadFile(codexPath)
		require.NoError(t, json.Unmarshal(raw, &result))

		ctx := result["required_context"].([]any)
		count := 0
		for _, v := range ctx {
			if v == ".strategist/agent-protocol.md" {
				count++
			}
		}
		assert.Equal(t, 1, count, "agent-protocol.md should appear exactly once")
	})

	t.Run("skips codex when file is not valid JSON", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		codexPath := filepath.Join(dir, ".sdd", "seedlings", "codex.seed.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(codexPath), 0o755))
		require.NoError(t, os.WriteFile(codexPath, []byte("not json"), 0o644))

		err := agentAwareness(dir)
		require.NoError(t, err)
	})

	t.Run("idempotent on second run", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		codexPath := filepath.Join(dir, ".sdd", "seedlings", "codex.seed.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(codexPath), 0o755))
		initial := map[string]any{
			"auto_activate":    true,
			"required_context": []any{".sdd/metadata.json"},
		}
		data, _ := json.Marshal(initial)
		require.NoError(t, os.WriteFile(codexPath, data, 0o644))

		require.NoError(t, agentAwareness(dir))
		first, _ := os.ReadFile(codexPath)

		require.NoError(t, agentAwareness(dir))
		second, _ := os.ReadFile(codexPath)

		assert.Equal(t, string(first), string(second), "second run must produce identical output")
	})

	t.Run("stable serialization between runs", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		codexPath := filepath.Join(dir, ".sdd", "seedlings", "codex.seed.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(codexPath), 0o755))
		initial := map[string]any{
			"z_last_key":       "z",
			"a_first_key":      "a",
			"required_context": []any{".sdd/metadata.json"},
			"middle_key":       true,
		}
		data, _ := json.Marshal(initial)
		require.NoError(t, os.WriteFile(codexPath, data, 0o644))

		require.NoError(t, agentAwareness(dir))
		run1, _ := os.ReadFile(codexPath)

		// Reset and run again — output must be byte-identical.
		require.NoError(t, os.WriteFile(codexPath, data, 0o644))
		require.NoError(t, agentAwareness(dir))
		run2, _ := os.ReadFile(codexPath)

		assert.Equal(t, string(run1), string(run2), "serialization must be stable across independent runs")
	})
}
