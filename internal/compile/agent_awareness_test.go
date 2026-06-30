package compile_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentAwareness(t *testing.T) {
	t.Parallel()

	t.Run("no-op when no agent files present", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		err := compile.AgentAwareness(dir)
		require.NoError(t, err)
	})

	t.Run("upserts antigravity section when file exists", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		agPath := filepath.Join(dir, ".antigravity", "antigravity-instructions.md")
		require.NoError(t, os.MkdirAll(filepath.Dir(agPath), 0o755))
		initial := "# Antigravity Instructions\n\n## Strategist Runtime Discovery\n\nOld content here.\n\n## Other Section\n\nKeep this.\n"
		require.NoError(t, os.WriteFile(agPath, []byte(initial), 0o644))

		require.NoError(t, compile.AgentAwareness(dir))

		content, _ := os.ReadFile(agPath)
		s := string(content)
		assert.Contains(t, s, "## Strategist Runtime Discovery")
		assert.NotContains(t, s, "Old content here.")
		assert.Contains(t, s, "strategist check")
		assert.Contains(t, s, "agent-protocol.md")
		assert.Contains(t, s, "## Other Section")
		assert.Contains(t, s, "Keep this.")
		assert.Contains(t, s, "Route selection is internal to Strategist")
		assert.Contains(t, s, "Strategist produces analysis, documentation, and handoff artifacts")
	})

	t.Run("appends strategist section to antigravity when section absent", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		agPath := filepath.Join(dir, ".antigravity", "antigravity-instructions.md")
		require.NoError(t, os.MkdirAll(filepath.Dir(agPath), 0o755))
		initial := "# Antigravity Instructions\n\n## Other Section\n\nContent.\n"
		require.NoError(t, os.WriteFile(agPath, []byte(initial), 0o644))

		require.NoError(t, compile.AgentAwareness(dir))

		content, _ := os.ReadFile(agPath)
		s := string(content)
		assert.Contains(t, s, "## Strategist Runtime Discovery")
		assert.Contains(t, s, "agent-protocol.md")
		assert.Contains(t, s, "## Other Section")
	})

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

		require.NoError(t, compile.AgentAwareness(dir))

		var result map[string]any
		raw, _ := os.ReadFile(codexPath)
		require.NoError(t, json.Unmarshal(raw, &result))

		ctx := result["required_context"].([]any)
		assert.Equal(t, ".strategist/agent-protocol.md", ctx[0], "agent-protocol.md must be first")
		assert.Contains(t, ctx, ".sdd/metadata.json")
		assert.Contains(t, ctx, ".strategist/SKILL.md")
		assert.NotNil(t, result["on_strategist_invoke"])
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

		require.NoError(t, compile.AgentAwareness(dir))

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

		err := compile.AgentAwareness(dir)
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

		require.NoError(t, compile.AgentAwareness(dir))
		first, _ := os.ReadFile(codexPath)

		require.NoError(t, compile.AgentAwareness(dir))
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

		require.NoError(t, compile.AgentAwareness(dir))
		run1, _ := os.ReadFile(codexPath)

		// Reset and run again — output must be byte-identical.
		require.NoError(t, os.WriteFile(codexPath, data, 0o644))
		require.NoError(t, compile.AgentAwareness(dir))
		run2, _ := os.ReadFile(codexPath)

		assert.Equal(t, string(run1), string(run2), "serialization must be stable across independent runs")
	})

	t.Run("upserts claude-instructions section when file exists", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		claudePath := filepath.Join(dir, ".claude", "claude-instructions.md")
		require.NoError(t, os.MkdirAll(filepath.Dir(claudePath), 0o755))
		initial := "# Claude Instructions\n\n## My Section\n\nKeep this.\n"
		require.NoError(t, os.WriteFile(claudePath, []byte(initial), 0o644))

		require.NoError(t, compile.AgentAwareness(dir))

		content, _ := os.ReadFile(claudePath)
		s := string(content)
		assert.Contains(t, s, "## Strategist Runtime Discovery")
		assert.Contains(t, s, "strategist check")
		assert.Contains(t, s, "agent-protocol.md")
		assert.Contains(t, s, "## My Section")
		assert.Contains(t, s, "Keep this.")
	})

	t.Run("replaces existing claude-instructions section on second run", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		claudePath := filepath.Join(dir, ".claude", "claude-instructions.md")
		require.NoError(t, os.MkdirAll(filepath.Dir(claudePath), 0o755))
		initial := "# Claude Instructions\n\n## Strategist Runtime Discovery\n\nOld awareness.\n\n## Other\n\nKeep.\n"
		require.NoError(t, os.WriteFile(claudePath, []byte(initial), 0o644))

		require.NoError(t, compile.AgentAwareness(dir))

		content, _ := os.ReadFile(claudePath)
		s := string(content)
		assert.NotContains(t, s, "Old awareness.")
		assert.Contains(t, s, "strategist check")
		assert.Contains(t, s, "## Other")
		assert.Contains(t, s, "Keep.")
	})

	t.Run("upserts codex commands section when file exists", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		codexCmdPath := filepath.Join(dir, ".codex", "commands.md")
		require.NoError(t, os.MkdirAll(filepath.Dir(codexCmdPath), 0o755))
		initial := "# Codex Commands\n\n## Run Tests\n\ngo test ./...\n"
		require.NoError(t, os.WriteFile(codexCmdPath, []byte(initial), 0o644))

		require.NoError(t, compile.AgentAwareness(dir))

		content, _ := os.ReadFile(codexCmdPath)
		s := string(content)
		assert.Contains(t, s, "## Strategist Runtime Discovery")
		assert.Contains(t, s, "strategist check")
		assert.Contains(t, s, "agent-protocol.md")
		assert.Contains(t, s, "## Run Tests")
		assert.Contains(t, s, "go test ./...")
	})

	t.Run("codex commands section is idempotent", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		codexCmdPath := filepath.Join(dir, ".codex", "commands.md")
		require.NoError(t, os.MkdirAll(filepath.Dir(codexCmdPath), 0o755))
		require.NoError(t, os.WriteFile(codexCmdPath, []byte("# Commands\n"), 0o644))

		require.NoError(t, compile.AgentAwareness(dir))
		first, _ := os.ReadFile(codexCmdPath)

		require.NoError(t, compile.AgentAwareness(dir))
		second, _ := os.ReadFile(codexCmdPath)

		assert.Equal(t, string(first), string(second), "second run must produce identical output")
	})

	t.Run("error in one agent does not block others", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		// Create codex seed with invalid JSON so it will fail.
		codexPath := filepath.Join(dir, ".sdd", "seedlings", "codex.seed.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(codexPath), 0o755))
		require.NoError(t, os.WriteFile(codexPath, []byte("not json"), 0o644))

		// Create a valid claude-instructions.md.
		claudePath := filepath.Join(dir, ".claude", "claude-instructions.md")
		require.NoError(t, os.MkdirAll(filepath.Dir(claudePath), 0o755))
		require.NoError(t, os.WriteFile(claudePath, []byte("# Claude\n"), 0o644))

		// AgentAwareness must not error even though codex failed.
		require.NoError(t, compile.AgentAwareness(dir))

		// Claude file must still have been updated.
		content, _ := os.ReadFile(claudePath)
		assert.Contains(t, string(content), "## Strategist Runtime Discovery")
	})
}

func TestRefreshAgentAwareness(t *testing.T) {
	t.Parallel()

	t.Run("returns false when agent-protocol generation fails (non-blocking)", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		// Non-existent strategistRoot — AgentProtocol will warn but not error.
		ok := compile.RefreshAgentAwareness(filepath.Join(dir, "no-such"), dir, "1.0.0", nil)
		assert.False(t, ok, "should return false when agent-protocol cannot be written")
	})

	t.Run("updates per-agent files when present regardless of protocol result", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		agPath := filepath.Join(dir, ".antigravity", "antigravity-instructions.md")
		require.NoError(t, os.MkdirAll(filepath.Dir(agPath), 0o755))
		require.NoError(t, os.WriteFile(agPath, []byte("# AG\n"), 0o644))

		// strategistRoot missing → protocol fails (returns false), but AgentAwareness still runs.
		ok := compile.RefreshAgentAwareness(filepath.Join(dir, "no-root"), dir, "2.0.0", nil)
		assert.False(t, ok, "protocol generation should have failed")

		content, _ := os.ReadFile(agPath)
		assert.Contains(t, string(content), "## Strategist Runtime Discovery")
	})
}
