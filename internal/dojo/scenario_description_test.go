package dojo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeCriteria(t *testing.T, dojoDir, scenario, body string) {
	t.Helper()
	dir := filepath.Join(dojoDir, scenario)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "criteria.yaml"), []byte(body), 0o644))
}

func TestScenarioDescription_ReturnsTheDescription(t *testing.T) {
	dojoDir := t.TempDir()
	writeCriteria(t, dojoDir, "sample", "scenario: sample\ndescription: \"sample scenario test\"\n")

	assert.Equal(t, "sample scenario test", ScenarioDescription(dojoDir, "sample"))
}

func TestScenarioDescription_MissingCriteriaFileIsEmpty(t *testing.T) {
	dojoDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dojoDir, "empty-scenario"), 0o755))

	assert.Empty(t, ScenarioDescription(dojoDir, "empty-scenario"))
}

func TestScenarioDescription_InvalidYAMLIsEmpty(t *testing.T) {
	dojoDir := t.TempDir()
	writeCriteria(t, dojoDir, "broken-scenario", "scenario: [unterminated\n")

	assert.Empty(t, ScenarioDescription(dojoDir, "broken-scenario"))
}

func TestScenarioDescription_NoDescriptionFieldIsEmpty(t *testing.T) {
	dojoDir := t.TempDir()
	writeCriteria(t, dojoDir, "plain", "scenario: plain\n")

	assert.Empty(t, ScenarioDescription(dojoDir, "plain"))
}
