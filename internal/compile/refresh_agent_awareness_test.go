package compile_test

// TestRefreshAgentAwareness, split out of agent_awareness_test.go. Stays
// package compile_test since RefreshAgentAwareness remains exported.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const refreshAwarenessTemplate = `---
generated_by: strategist compile
version: {{.Version}}
generated_at: {{.GeneratedAt}}
---

Discovery slot: {{.Slots.Discovery}}
`

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

	t.Run("returns true when agent-protocol generation succeeds", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		testutil.MinimalRoot(t, dir)

		ok := compile.RefreshAgentAwareness(dir, dir, "1.0.0", []byte(refreshAwarenessTemplate))
		assert.True(t, ok, "should return true when agent-protocol.md was written")
		assert.FileExists(t, filepath.Join(dir, "agent-protocol.md"))
	})

	t.Run("returns false when agent-protocol generation errors with non-empty template", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		// strategistRoot has no active.yaml, so agentProtocol fails despite a non-empty template.
		ok := compile.RefreshAgentAwareness(dir, dir, "1.0.0", []byte(refreshAwarenessTemplate))
		assert.False(t, ok, "should return false when agent-protocol.md write fails")
	})
}
