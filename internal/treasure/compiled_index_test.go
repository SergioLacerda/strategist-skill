package treasure

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeIndexGz(t *testing.T, path string, compiledAt int64, sourceIDs ...string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	require.NoError(t, err)
	gz := gzip.NewWriter(f)
	meta := make(map[string]any)
	for _, id := range sourceIDs {
		meta[id] = map[string]any{"id": id}
	}
	require.NoError(t, json.NewEncoder(gz).Encode(map[string]any{
		"compiled_at": compiledAt,
		"source_meta": meta,
	}))
	require.NoError(t, gz.Close())
	require.NoError(t, f.Close())
}

func TestLoadCompiledIndex_NotExist(t *testing.T) {
	t.Parallel()
	ids, ts, err := LoadCompiledIndex(t.TempDir())
	require.NoError(t, err)
	assert.Nil(t, ids)
	assert.Zero(t, ts)
}

func TestLoadCompiledIndex_CorruptGzip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ".compiled", ".index.gz")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("not gzip"), 0o644))

	_, _, err := LoadCompiledIndex(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decompress")
}

func TestLoadCompiledIndex_OpenErrorPropagates(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	compiledDir := filepath.Join(dir, ".compiled")
	require.NoError(t, os.MkdirAll(compiledDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(compiledDir, ".index.gz"), []byte("x"), 0o644))
	require.NoError(t, os.Chmod(compiledDir, 0o000))
	t.Cleanup(func() { _ = os.Chmod(compiledDir, 0o755) })

	_, _, err := LoadCompiledIndex(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "open .compiled/.index.gz")
}

func TestLoadCompiledIndex_DecodeErrorPropagates(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ".compiled", ".index.gz")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	require.NoError(t, err)
	gz := gzip.NewWriter(f)
	_, err = gz.Write([]byte("not json"))
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	require.NoError(t, f.Close())

	_, _, err = LoadCompiledIndex(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode .compiled/.index.gz")
}

func TestLoadCompiledIndex_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ".compiled", ".index.gz")
	writeIndexGz(t, path, 1700000000, "chest-a", "chest-b")

	ids, ts, err := LoadCompiledIndex(dir)
	require.NoError(t, err)
	assert.EqualValues(t, 1700000000, ts)
	assert.True(t, ids["chest-a"])
	assert.True(t, ids["chest-b"])
	assert.False(t, ids["chest-c"])
}
