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

func TestCompileIndex(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		wantErr bool
		check   func(t *testing.T, artifact map[string]any)
	}{
		{
			name:    "empty sources",
			content: "sources: []\n",
			check: func(t *testing.T, a map[string]any) {
				assert.Equal(t, "strategist-compiled-index/1.0", a["schema"])
				assertSourceStats(t, a)
				tags := a["tags"].(map[string]any)
				assert.Empty(t, tags)
			},
		},
		{
			name: "single source with tags builds inverted index",
			content: `sources:
  - id: arch-doc
    tags: [architecture, system-design]
`,
			check: func(t *testing.T, a map[string]any) {
				tags := a["tags"].(map[string]any)
				sourceMeta := a["source_meta"].(map[string]any)
				assert.Contains(t, sourceMeta, "arch-doc")
				assert.Contains(t, tags, "architecture")
				assert.Contains(t, tags, "system-design")
				archIDs := tags["architecture"].([]any)
				assert.Contains(t, archIDs, "arch-doc")
			},
		},
		{
			name: "multiple sources sharing a tag",
			content: `sources:
  - id: doc-a
    tags: [shared]
  - id: doc-b
    tags: [shared, unique]
`,
			check: func(t *testing.T, a map[string]any) {
				tags := a["tags"].(map[string]any)
				sharedIDs := tags["shared"].([]any)
				assert.Len(t, sharedIDs, 2)
				assert.Contains(t, sharedIDs, "doc-a")
				assert.Contains(t, sharedIDs, "doc-b")
			},
		},
		{
			name:    "missing knowledge index file returns error",
			content: "",
			wantErr: true,
		},
		{
			name:    "invalid YAML in knowledge index returns error",
			content: "sources: [invalid: yaml: content",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			kiPath, out := prepareCompileIndexCase(t, tt.wantErr, tt.content)
			err := compile.Index(kiPath, out)
			assertCompileIndexResult(t, out, err, tt.wantErr, tt.check)
		})
	}
}

func prepareCompileIndexCase(t *testing.T, wantErr bool, content string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	out := filepath.Join(dir, ".compiled", ".index.gz")
	if wantErr {
		return filepath.Join(dir, "nonexistent.yaml"), out
	}
	kiPath := filepath.Join(dir, "knowledge.index.yaml")
	require.NoError(t, os.WriteFile(kiPath, []byte(content), 0o644))
	return kiPath, out
}

func assertCompileIndexResult(t *testing.T, out string, err error, wantErr bool, check func(*testing.T, map[string]any)) {
	t.Helper()
	if wantErr {
		require.Error(t, err)
		return
	}
	require.NoError(t, err)
	require.FileExists(t, out)
	var artifact map[string]any
	testutil.ReadGzJSON(t, out, &artifact)
	if check != nil {
		check(t, artifact)
	}
}

func TestCompileIndex_InvalidYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	kiPath := filepath.Join(dir, "knowledge.index.yaml")
	require.NoError(t, os.WriteFile(kiPath, []byte("sources: [unclosed bracket"), 0o644))
	err := compile.Index(kiPath, filepath.Join(dir, ".compiled", ".index.gz"))
	require.Error(t, err)
}

func TestCompileIndex_WriteError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	kiPath := filepath.Join(dir, "knowledge.index.yaml")
	require.NoError(t, os.WriteFile(kiPath, []byte("sources: []\n"), 0o644))
	// Output path is a directory, so the final gzip write must fail.
	outPath := filepath.Join(dir, ".index.gz")
	require.NoError(t, os.MkdirAll(outPath, 0o755))
	err := compile.Index(kiPath, outPath)
	require.Error(t, err)
	assert.ErrorContains(t, err, "compile index: write")
}
