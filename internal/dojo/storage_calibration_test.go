package dojo_test

import (
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/dojo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReservedNames_AreNotScenarioNames(t *testing.T) {
	for _, name := range dojo.ReservedNames() {
		_, err := dojo.NewStoragePaths(t.TempDir(), name)
		require.Error(t, err, name)
		assert.True(t, dojo.IsReservedName(name))
	}
	assert.False(t, dojo.IsReservedName("treasure-chest"))
}

func TestCalibrationArtifactPolicies_ExtendWithoutChangingTheScenarioContract(t *testing.T) {
	assert.Len(t, dojo.ArtifactPolicies(), 5)
	assert.Len(t, dojo.CalibrationArtifactPolicies(), 8)
	assert.Len(t, dojo.AllArtifactPolicies(), 13)
	for _, policy := range dojo.CalibrationArtifactPolicies() {
		assert.NotEmpty(t, policy.Durability)
		assert.NotEmpty(t, policy.Recovery)
		assert.NotEmpty(t, policy.Owner)
	}
}

func TestNewCasePaths_DefaultsToTheCaseDirectory(t *testing.T) {
	base := t.TempDir()
	paths, err := dojo.NewCasePaths(base, "jev-handoff", "")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(base, "dojo", "jev-handoff"), paths.Output)
	assert.Equal(t, filepath.Join(base, "dojo", "jev-handoff", "observations.jsonl"), paths.ObservationsPath)
	assert.Equal(t, filepath.Join(base, "dojo", dojo.DirCalibrations), paths.CalibrationsDir)
}

func TestNewCasePaths_ConfiguredOutputIsValidated(t *testing.T) {
	base := t.TempDir()
	paths, err := dojo.NewCasePaths(base, "jev-handoff", "dojo/out/jev")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(base, "dojo", "out", "jev"), paths.Output)
	absoluteOutput, err := filepath.Abs(filepath.Join("abs", "path"))
	require.NoError(t, err)

	for _, bad := range []string{"../outside", absoluteOutput, "refined/x", "done", "pending/y", ".strategist/z"} {
		_, err := dojo.NewCasePaths(base, "jev-handoff", bad)
		require.Error(t, err, bad)
	}
}
