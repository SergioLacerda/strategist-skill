// Package testutil provides shared test helpers for the strategist-skill module.
// Import this package in test files using the standard Go test helper pattern.
package testutil

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// ValidMinimalPersonaYAML returns the smallest persona fixture accepted by runtime validation.
func ValidMinimalPersonaYAML() []byte {
	return []byte(`id: epic
tone_directive: be precise
phase_labels:
  discovery: analysis
  refinement: refinement
  execution: execution
diagnostics:
  pipeline_header: "[Strategist] pipeline=starting mission_id={id}"
  bootstrap_origin: "[Strategist] profile_path={path} active_yaml={active} reason={reason}"
`)
}

// WriteGzJSON writes v as gzip-compressed JSON to path, creating parent dirs as needed.
func WriteGzJSON(t testing.TB, path string, v any) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path) //nolint:gosec // G304: test helper writes explicit temporary fixture paths
	require.NoError(t, err)
	gz := gzip.NewWriter(f)
	require.NoError(t, json.NewEncoder(gz).Encode(v))
	require.NoError(t, gz.Close())
	require.NoError(t, f.Close())
}

// ReadGzJSON decompresses a gzipped JSON artifact at path into v.
func ReadGzJSON(t testing.TB, path string, v any) {
	t.Helper()
	f, err := os.Open(path) //nolint:gosec // G304: test helper reads explicit temporary fixture paths
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
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte("mode: full\nbase_path: .analysis\nslots:\n  discovery: brainstorming\n  refinement: openspec-explore\n  execution: sdd-ask\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "personas", "epic.yaml"), ValidMinimalPersonaYAML(), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "roles", "default.yaml"), []byte("name: Default\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.yaml"), []byte("load_always: []\nload_by_task_type: {}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "knowledge.index.yaml"), []byte("sources: []\n"), 0o644))
}

// RequirePOSIXShell skips a test that fakes an executable with a #!/bin/sh
// script. Windows cannot exec such a script (no PATHEXT match, no /bin/sh), so
// the real exec path is covered there by the standalone payload smoke instead.
func RequirePOSIXShell(t *testing.T) {
	t.Helper()
	skipOnWindows(t, runtime.GOOS, "fakes an executable with a POSIX shell script; covered on Windows by the standalone payload smoke")
}

// SkipOnWindowsReadDirOfFile skips a test that relies on os.ReadDir failing
// for a regular file: on Windows that is reported as "path not found", which
// the callers deliberately treat as an absent directory.
func SkipOnWindowsReadDirOfFile(t *testing.T) {
	t.Helper()
	skipOnWindows(t, runtime.GOOS, "os.ReadDir on a regular file reports not-exist on Windows, which callers tolerate by design")
}

// SetHome points the home-directory lookup at dir on every OS (HOME on Unix,
// USERPROFILE on Windows). An empty dir makes the lookup fail everywhere.
func SetHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
}

func skipOnWindows(t *testing.T, goos, reason string) {
	t.Helper()
	if goos == "windows" {
		t.Skip(reason)
	}
}
