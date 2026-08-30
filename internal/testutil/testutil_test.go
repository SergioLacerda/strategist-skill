package testutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestValidMinimalPersonaYAML_IsWellFormedYAML(t *testing.T) {
	t.Parallel()

	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(testutil.ValidMinimalPersonaYAML(), &doc))
	assert.Equal(t, "epic", doc["id"])
	assert.Contains(t, doc, "phase_labels")
	assert.Contains(t, doc, "diagnostics")
}

func TestWriteGzJSONThenReadGzJSON_RoundTrips(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "artifact.gz")

	type payload struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	want := payload{Name: "strategist", Count: 3}

	testutil.WriteGzJSON(t, path, want)
	require.FileExists(t, path)

	var got payload
	testutil.ReadGzJSON(t, path, &got)
	assert.Equal(t, want, got)
}

func TestMinimalRoot_WritesExpectedRuntimeFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)

	for _, rel := range []string{
		"active.yaml",
		"personas/epic.yaml",
		"roles/default.yaml",
		"index.yaml",
		"knowledge.index.yaml",
	} {
		t.Run(rel, func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(filepath.Join(dir, rel)) //nolint:gosec // G304: test reads a path under t.TempDir(), not user input
			require.NoError(t, err)
			assert.NotEmpty(t, data)
		})
	}

	personaData, err := os.ReadFile(filepath.Join(dir, "personas", "epic.yaml")) //nolint:gosec // G304: test reads a path under t.TempDir(), not user input
	require.NoError(t, err)
	assert.Equal(t, testutil.ValidMinimalPersonaYAML(), personaData)
}
