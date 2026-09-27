package handoff

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadArtifact(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		_, err := readArtifact(filepath.Join(t.TempDir(), "missing.md"))
		require.ErrorContains(t, err, "read artifact")
	})
	t.Run("empty file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "empty.md")
		require.NoError(t, os.WriteFile(path, []byte("   \n"), 0o600))
		_, err := readArtifact(path)
		require.ErrorContains(t, err, "artifact is empty")
	})
	t.Run("valid content", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "ok.md")
		require.NoError(t, os.WriteFile(path, []byte("content"), 0o600))
		content, err := readArtifact(path)
		require.NoError(t, err)
		require.Equal(t, "content", string(content))
	})
}

func TestParseFrontmatter(t *testing.T) {
	t.Run("missing frontmatter", func(t *testing.T) {
		_, _, err := parseFrontmatter([]byte("# no frontmatter\n"))
		require.ErrorContains(t, err, "frontmatter is missing")
	})
	t.Run("unclosed frontmatter", func(t *testing.T) {
		_, _, err := parseFrontmatter([]byte("---\nmission_id: m-1\n"))
		require.ErrorContains(t, err, "frontmatter is unclosed")
	})
	t.Run("malformed yaml", func(t *testing.T) {
		_, _, err := parseFrontmatter([]byte("---\nmission_id: [unterminated\n---\nbody\n"))
		require.ErrorContains(t, err, "parse frontmatter")
	})
	t.Run("valid frontmatter", func(t *testing.T) {
		frontmatter, body, err := parseFrontmatter([]byte("---\nmission_id: m-1\n---\nbody\n"))
		require.NoError(t, err)
		require.Equal(t, "m-1", frontmatter["mission_id"])
		require.Equal(t, "\nbody", string(body))
	})
}

func TestValidateIdentity(t *testing.T) {
	t.Run("missing mission_id", func(t *testing.T) {
		err := validateIdentity(map[string]any{}, "m-1", []string{"ranger_done"})
		require.ErrorContains(t, err, `mission_id ""`)
	})
	t.Run("mismatched mission_id", func(t *testing.T) {
		err := validateIdentity(map[string]any{"mission_id": "m-2"}, "m-1", []string{"ranger_done"})
		require.ErrorContains(t, err, `mission_id "m-2"`)
	})
	t.Run("status not allowed", func(t *testing.T) {
		err := validateIdentity(map[string]any{"mission_id": "m-1", "mission_status": "other"}, "m-1", []string{"ranger_done"})
		require.ErrorContains(t, err, `mission_status "other"`)
	})
	t.Run("valid", func(t *testing.T) {
		err := validateIdentity(map[string]any{"mission_id": "m-1", "mission_status": "ranger_done"}, "m-1", []string{"ranger_done"})
		require.NoError(t, err)
	})
}

func TestValidateTaskClassifications(t *testing.T) {
	t.Run("no tasks found", func(t *testing.T) {
		err := validateTaskClassifications([]byte("# Tasks\nnothing here\n"))
		require.ErrorContains(t, err, "no classified tasks found")
	})
	t.Run("unclassified task", func(t *testing.T) {
		err := validateTaskClassifications([]byte("- [ ] 1.1 change source\n"))
		require.ErrorContains(t, err, "no explicit classification")
	})
	t.Run("classified task", func(t *testing.T) {
		err := validateTaskClassifications([]byte("- [ ] 1.1 [implementation_handoff] change source\n"))
		require.NoError(t, err)
	})
}
