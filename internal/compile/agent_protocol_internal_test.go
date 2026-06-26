package compile

// Whitebox tests for unexported helpers in agent_protocol.go and agent_awareness.go.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	err := writeFile(filepath.Join(dir, "out.md"), "content")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent protocol: write")
}

// --- marshalSortedJSON ---

func TestMarshalSortedJSON_NonSerializableValue(t *testing.T) {
	t.Parallel()
	_, err := marshalSortedJSON(map[string]any{"key": make(chan int)})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshal value for key")
}

func TestMarshalSortedJSON_EmptyMap(t *testing.T) {
	t.Parallel()
	out, err := marshalSortedJSON(map[string]any{})
	require.NoError(t, err)
	assert.Equal(t, "{}", string(out))
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

// --- upsertCodexSeed write error ---

func TestUpsertCodexSeed_WriteError(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "codex.seed.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"required_context":[]}`), 0o644))
	require.NoError(t, os.Chmod(path, 0o444))
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	err := upsertCodexSeed(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "codex seed: write")
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

// --- AgentAwareness slog.Warn path (write error on antigravity) ---

func TestAgentAwareness_AntigravityWriteErrorIsNonBlocking(t *testing.T) {
	t.Parallel()
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	agDir := filepath.Join(dir, ".antigravity")
	require.NoError(t, os.MkdirAll(agDir, 0o755))
	agPath := filepath.Join(agDir, "antigravity-instructions.md")
	require.NoError(t, os.WriteFile(agPath, []byte("# Antigravity\n"), 0o644))
	require.NoError(t, os.Chmod(agPath, 0o444))
	t.Cleanup(func() { _ = os.Chmod(agPath, 0o644) })

	err := AgentAwareness(dir)
	require.NoError(t, err, "AgentAwareness must always return nil even when per-file update fails")
}
