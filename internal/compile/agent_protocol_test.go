package compile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const minimalAgentProtocolTemplate = `---
generated_by: strategist compile
version: {{.Version}}
generated_at: {{.GeneratedAt}}
path_model: runtime-only
---

# Strategist Agent Protocol

Discovery slot: {{.Slots.Discovery}}
Refinement slot: {{.Slots.Refinement}}
Execution slot: {{.Slots.Execution}}
`

func TestAgentProtocol(t *testing.T) {
	t.Parallel()

	t.Run("creates file when absent", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		testutil.MinimalRoot(t, dir)

		err := compile.AgentProtocol(dir, []byte(minimalAgentProtocolTemplate), "1.0.0")
		require.NoError(t, err)

		out := filepath.Join(dir, "agent-protocol.md")
		require.FileExists(t, out)
		content, _ := os.ReadFile(out)
		s := string(content)
		assert.Contains(t, s, "version: 1.0.0")
		assert.Contains(t, s, "Discovery slot: brainstorming")
		assert.Contains(t, s, "Refinement slot: openspec-explore")
		assert.Contains(t, s, "Execution slot: sdd-ask")
	})

	t.Run("upserts body when file exists", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		testutil.MinimalRoot(t, dir)

		out := filepath.Join(dir, "agent-protocol.md")
		initial := "---\ngenerated_by: strategist compile\nversion: 0.9.0\ngenerated_at: 2026-01-01T00:00:00Z\npath_model: runtime-only\n---\n\nOld body content.\n"
		require.NoError(t, os.WriteFile(out, []byte(initial), 0o644))

		require.NoError(t, compile.AgentProtocol(dir, []byte(minimalAgentProtocolTemplate), "1.1.0"))

		content, _ := os.ReadFile(out)
		s := string(content)
		assert.Contains(t, s, "version: 1.1.0")
		assert.NotContains(t, s, "Old body content.")
		assert.Contains(t, s, "Discovery slot: brainstorming")
	})

	t.Run("preserves extra frontmatter fields on upsert", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		testutil.MinimalRoot(t, dir)

		out := filepath.Join(dir, "agent-protocol.md")
		initial := "---\ngenerated_by: strategist compile\nversion: 0.9.0\ngenerated_at: 2026-01-01T00:00:00Z\npath_model: runtime-only\ncustom_note: keep-me\n---\n\nOld body.\n"
		require.NoError(t, os.WriteFile(out, []byte(initial), 0o644))

		require.NoError(t, compile.AgentProtocol(dir, []byte(minimalAgentProtocolTemplate), "1.2.0"))

		content, _ := os.ReadFile(out)
		s := string(content)
		assert.Contains(t, s, "custom_note: keep-me")
		assert.Contains(t, s, "version: 1.2.0")
		assert.NotContains(t, s, "Old body.")
	})

	t.Run("overwrites malformed frontmatter entirely", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		testutil.MinimalRoot(t, dir)

		out := filepath.Join(dir, "agent-protocol.md")
		require.NoError(t, os.WriteFile(out, []byte("no frontmatter here\njust body"), 0o644))

		require.NoError(t, compile.AgentProtocol(dir, []byte(minimalAgentProtocolTemplate), "1.0.0"))

		content, _ := os.ReadFile(out)
		assert.Contains(t, string(content), "generated_by: strategist compile")
	})

	t.Run("error when active.yaml missing", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		err := compile.AgentProtocol(dir, []byte(minimalAgentProtocolTemplate), "1.0.0")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "agent protocol")
	})

	t.Run("error on invalid template syntax", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		testutil.MinimalRoot(t, dir)

		err := compile.AgentProtocol(dir, []byte("{{.Invalid syntax"), "1.0.0")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "agent protocol")
	})
}

func TestAgentProtocol_GeneratedAtIsSet(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)

	require.NoError(t, compile.AgentProtocol(dir, []byte(minimalAgentProtocolTemplate), "1.0.0"))

	out := filepath.Join(dir, "agent-protocol.md")
	content, _ := os.ReadFile(out)
	assert.Contains(t, string(content), "generated_at: 20", "generated_at should be set to a current timestamp")
}
