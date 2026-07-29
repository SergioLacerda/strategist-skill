package compile

// Whitebox tests for unexported helpers in agent_protocol.go and agent_awareness.go.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- agentProtocol ---
// Relocated from agent_protocol_test.go (formerly package compile_test) when
// AgentProtocol was unexported to agentProtocol.

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

		err := agentProtocol(dir, []byte(minimalAgentProtocolTemplate), "1.0.0")
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

		require.NoError(t, agentProtocol(dir, []byte(minimalAgentProtocolTemplate), "1.1.0"))

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

		require.NoError(t, agentProtocol(dir, []byte(minimalAgentProtocolTemplate), "1.2.0"))

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

		require.NoError(t, agentProtocol(dir, []byte(minimalAgentProtocolTemplate), "1.0.0"))

		content, _ := os.ReadFile(out)
		assert.Contains(t, string(content), "generated_by: strategist compile")
	})

	t.Run("error when active.yaml missing", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		err := agentProtocol(dir, []byte(minimalAgentProtocolTemplate), "1.0.0")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "agent protocol")
	})

	t.Run("error on invalid template syntax", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		testutil.MinimalRoot(t, dir)

		err := agentProtocol(dir, []byte("{{.Invalid syntax"), "1.0.0")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "agent protocol")
	})

	t.Run("error when template references a field that does not exist", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		testutil.MinimalRoot(t, dir)

		// Parses fine (valid syntax), but Slots has no "Bogus" field — fails at Execute time.
		err := agentProtocol(dir, []byte("{{.Slots.Bogus}}"), "1.0.0")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "render template")
	})

	t.Run("error when existing agent-protocol.md path is a directory", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		testutil.MinimalRoot(t, dir)
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "agent-protocol.md"), 0o755))

		err := agentProtocol(dir, []byte(minimalAgentProtocolTemplate), "1.0.0")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "agent protocol: read")
	})

	t.Run("overwrites entirely when closing frontmatter marker is missing", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		testutil.MinimalRoot(t, dir)

		out := filepath.Join(dir, "agent-protocol.md")
		require.NoError(t, os.WriteFile(out, []byte("---\nversion: 0.9.0\nno closing marker here"), 0o644))

		require.NoError(t, agentProtocol(dir, []byte(minimalAgentProtocolTemplate), "1.0.0"))

		content, _ := os.ReadFile(out)
		assert.Contains(t, string(content), "version: 1.0.0")
		assert.NotContains(t, string(content), "no closing marker here")
	})
}

func TestAgentProtocol_GeneratedAtIsSet(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)

	require.NoError(t, agentProtocol(dir, []byte(minimalAgentProtocolTemplate), "1.0.0"))

	out := filepath.Join(dir, "agent-protocol.md")
	content, _ := os.ReadFile(out)
	assert.Contains(t, string(content), "generated_at: 20", "generated_at should be set to a current timestamp")
}

// --- renderedBodyOnly ---

func TestRenderedBodyOnly_NoOpenMarker(t *testing.T) {
	t.Parallel()
	in := "no frontmatter here\njust body"
	assert.Equal(t, in, renderedBodyOnly(in))
}

func TestRenderedBodyOnly_NoCloseMarker(t *testing.T) {
	t.Parallel()
	in := "---\nversion: 1.0\nno closing marker after this"
	assert.Equal(t, in, renderedBodyOnly(in))
}

func TestRenderedBodyOnly_ExtractsBody(t *testing.T) {
	t.Parallel()
	in := "---\nversion: 1.0\n---\nbody content\n"
	assert.Equal(t, "body content\n", renderedBodyOnly(in))
}

// --- replaceYAMLLine ---

func TestReplaceYAMLLine_KeyNotFound(t *testing.T) {
	t.Parallel()
	s := "version: 1.0\nother: value\n"
	result := replaceYAMLLine(s, "missing:", "new")
	assert.Equal(t, s, result)
}

func TestReplaceYAMLLine_ReplacesFirst(t *testing.T) {
	t.Parallel()
	s := "---\nversion: old\ngenerated_at: 2026-01-01\n---\n"
	result := replaceYAMLLine(s, "version:", "new")
	assert.Contains(t, result, "version: new")
	assert.NotContains(t, result, "version: old")
}

// --- writeFile ---

func TestWriteFile_Error(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o444))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	err := writeFile(filepath.Join(dir, "out.md"), []byte("content"), "agent protocol")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent protocol: write")
}

// --- upsertSection write error ---

func TestUpsertSection_WriteError(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "instructions.md")
	require.NoError(t, os.WriteFile(path, []byte("# Instructions\n"), 0o644))
	require.NoError(t, os.Chmod(path, 0o444))
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	err := upsertSection(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upsert section: write")
}

func TestUpsertSection_ReadError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// A directory at the target path makes os.ReadFile fail with a non-permission error.
	path := filepath.Join(dir, "instructions.md")
	require.NoError(t, os.MkdirAll(path, 0o755))

	err := upsertSection(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upsert section: read")
}

func TestAppendMarkdownSection_AddsMissingTrailingNewline(t *testing.T) {
	t.Parallel()
	// Content without a trailing newline must gain one before the section is appended.
	got := appendMarkdownSection("# Header\n\nNo trailing newline", "## New Section\n\nBody.")
	assert.Equal(t, "# Header\n\nNo trailing newline\n\n## New Section\n\nBody.\n", got)
}

func TestUpsertSection_AppendWhenAbsent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "instructions.md")
	require.NoError(t, os.WriteFile(path, []byte("# My Instructions\n"), 0o644))

	err := upsertSection(path)
	require.NoError(t, err)

	content, _ := os.ReadFile(path)
	assert.Contains(t, string(content), "## Strategist Runtime Discovery")
}

func TestUpsertSection_ReplaceExisting(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "instructions.md")
	initial := "# Header\n\n## Strategist Runtime Discovery\n\nOld content here.\n\n## Other Section\n\nKeep this.\n"
	require.NoError(t, os.WriteFile(path, []byte(initial), 0o644))

	err := upsertSection(path)
	require.NoError(t, err)

	content, _ := os.ReadFile(path)
	s := string(content)
	assert.Contains(t, s, "## Strategist Runtime Discovery")
	assert.NotContains(t, s, "Old content here.")
	assert.Contains(t, s, "## Other Section")
}

// --- markdownTail ---

func TestMarkdownTail_PreservesContentAfterDeeperHeading(t *testing.T) {
	t.Parallel()
	after := "\n\nSome section body.\n\n### Deeper Heading\n\nMore content.\n"
	tail := markdownTail(after)
	assert.Contains(t, tail, "### Deeper Heading")
	assert.Contains(t, tail, "More content.")
}

func TestMarkdownTail_LastSectionInFileReturnsEmpty(t *testing.T) {
	t.Parallel()
	after := "\n\nSome trailing content with no further heading.\n"
	assert.Empty(t, markdownTail(after))
}

// --- AgentAwareness slog.Warn path (write error on a seed target) ---

func TestAgentAwareness_SeedWriteErrorIsNonBlocking(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	require.NoError(t, os.MkdirAll(claudeDir, 0o755))
	claudePath := filepath.Join(claudeDir, "claude-instructions.md")
	require.NoError(t, os.WriteFile(claudePath, []byte("# Claude\n"), 0o644))
	require.NoError(t, os.Chmod(claudePath, 0o444))
	t.Cleanup(func() { _ = os.Chmod(claudePath, 0o644) })

	err := agentAwareness(dir)
	require.NoError(t, err, "agentAwareness must always return nil even when per-file update fails")
}
