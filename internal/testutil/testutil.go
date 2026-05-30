// Package testutil provides shared test helpers for the strategist-skill module.
// Import this package in test files using the standard Go test helper pattern.
package testutil

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// WriteGzJSON writes v as gzip-compressed JSON to path, creating parent dirs as needed.
func WriteGzJSON(t testing.TB, path string, v any) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	require.NoError(t, err)
	gz := gzip.NewWriter(f)
	require.NoError(t, json.NewEncoder(gz).Encode(v))
	require.NoError(t, gz.Close())
	require.NoError(t, f.Close())
}

// ReadGzJSON decompresses a gzipped JSON artifact at path into v.
func ReadGzJSON(t testing.TB, path string, v any) {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err, "open artifact %s", path)
	defer f.Close() //nolint:errcheck
	gz, err := gzip.NewReader(f)
	require.NoError(t, err, "gzip reader")
	defer gz.Close() //nolint:errcheck
	require.NoError(t, json.NewDecoder(gz).Decode(v), "json decode")
}

// MinimalRoot creates a minimal .strategist/-like directory tree in dir suitable
// for compile.Config, compile.Domain, compile.Index, and compile.All:
// active.yaml, personas/epic.yaml, roles/default.yaml, index.yaml, knowledge.index.yaml.
func MinimalRoot(t testing.TB, dir string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "personas"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "roles"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte("mode: full\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "personas", "epic.yaml"), []byte("name: Epic\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "roles", "default.yaml"), []byte("name: Default\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.yaml"), []byte("load_always: []\nload_by_task_type: {}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "knowledge.index.yaml"), []byte("sources: []\n"), 0o644))
}
