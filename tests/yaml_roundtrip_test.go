//go:build integration

package tests_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/compile"
	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func mustReadEmbed(t *testing.T, relPath string) []byte {
	t.Helper()
	data, err := embedpkg.Extractor{}.ReadFile(relPath)
	require.NoError(t, err, "ReadFile(%s)", relPath)
	require.NotEmpty(t, data, "embedded file %s is empty", relPath)
	return data
}

func TestRoundtrip_IndexYAML(t *testing.T) {
	t.Parallel()

	data := mustReadEmbed(t, "index.yaml")

	var s compile.DomainIndex
	require.NoError(t, yaml.Unmarshal(data, &s))

	// index.yaml has load_always with at least the core schema files
	assert.NotEmpty(t, s.LoadAlways, "load_always must not be empty after unmarshal")

	out, err := yaml.Marshal(s)
	require.NoError(t, err)

	var s2 compile.DomainIndex
	require.NoError(t, yaml.Unmarshal(out, &s2))

	assert.Equal(t, s.LoadAlways, s2.LoadAlways, "load_always survived round-trip")
	assert.Equal(t, s.LoadByTaskType, s2.LoadByTaskType, "load_by_task_type survived round-trip")
}

func TestRoundtrip_KnowledgeIndexYAML(t *testing.T) {
	t.Parallel()

	data := mustReadEmbed(t, "knowledge.index.yaml")

	var s compile.KnowledgeIndex
	require.NoError(t, yaml.Unmarshal(data, &s))

	// Default knowledge index ships with an empty sources list — verify the struct is valid
	assert.NotNil(t, s.Sources, "sources field must be parseable (nil means unmarshal failed)")

	out, err := yaml.Marshal(s)
	require.NoError(t, err)

	var s2 compile.KnowledgeIndex
	require.NoError(t, yaml.Unmarshal(out, &s2))

	assert.Equal(t, len(s.Sources), len(s2.Sources), "source count survived round-trip")
}

func TestRoundtrip_ActiveYAML(t *testing.T) {
	t.Parallel()

	data := mustReadEmbed(t, "templates/epic-standalone.yaml")

	var s compile.ActiveConfig
	require.NoError(t, yaml.Unmarshal(data, &s))

	assert.NotEmpty(t, s.Mode, "mode must survive unmarshal")
	assert.NotEmpty(t, s.BasePath, "base_path must survive unmarshal")
	assert.NotEmpty(t, s.Slots, "slots must survive unmarshal")

	out, err := yaml.Marshal(s)
	require.NoError(t, err)

	var s2 compile.ActiveConfig
	require.NoError(t, yaml.Unmarshal(out, &s2))

	assert.Equal(t, s.Mode, s2.Mode, "mode survived round-trip")
	assert.Equal(t, s.BasePath, s2.BasePath, "base_path survived round-trip")
	assert.Equal(t, s.Slots, s2.Slots, "slots survived round-trip")
	assert.Equal(t, s.AdrEnabled, s2.AdrEnabled, "adr_enabled survived round-trip")
}

func TestRoundtrip_PersonaEpic(t *testing.T) {
	t.Parallel()

	data := mustReadEmbed(t, "personas/epic.yaml")

	var s compile.PersonaConfig
	require.NoError(t, yaml.Unmarshal(data, &s))

	assert.NotEmpty(t, s.ID, "id must survive unmarshal")
	assert.NotEmpty(t, s.Description, "description must survive unmarshal")
	assert.NotEmpty(t, s.PhaseLabels.Discovery, "phase_labels.discovery must survive unmarshal")
	assert.NotEmpty(t, s.PhaseLabels.Refinement, "phase_labels.refinement must survive unmarshal")
	assert.NotEmpty(t, s.PhaseLabels.Execution, "phase_labels.execution must survive unmarshal")
	assert.NotEmpty(t, s.ProgressPrefix, "progress_prefix must survive unmarshal")

	out, err := yaml.Marshal(s)
	require.NoError(t, err)

	var s2 compile.PersonaConfig
	require.NoError(t, yaml.Unmarshal(out, &s2))

	assert.Equal(t, s.ID, s2.ID, "id survived round-trip")
	assert.Equal(t, s.PhaseLabels, s2.PhaseLabels, "phase_labels survived round-trip")
	assert.Equal(t, s.ProgressPrefix, s2.ProgressPrefix, "progress_prefix survived round-trip")
}

func TestRoundtrip_ApprovalGateContract(t *testing.T) {
	t.Parallel()

	data := mustReadEmbed(t, "contracts/approval-gate.yaml")

	var s compile.ApprovalGateContract
	require.NoError(t, yaml.Unmarshal(data, &s))

	assert.NotEmpty(t, s.Module, "module must survive unmarshal")
	assert.NotEmpty(t, s.Type, "type must survive unmarshal")

	out, err := yaml.Marshal(s)
	require.NoError(t, err)

	var s2 compile.ApprovalGateContract
	require.NoError(t, yaml.Unmarshal(out, &s2))

	assert.Equal(t, s.Module, s2.Module, "module survived round-trip")
	assert.Equal(t, s.Type, s2.Type, "type survived round-trip")
}
