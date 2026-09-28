package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The generation is bumped in the same commit that changes the runtime layout,
// so the value is pinned: changing it without changing this test is a review
// signal, not an accident.
func TestRuntimeLayoutGenerationIsPinned(t *testing.T) {
	assert.Equal(t, 1, domain.RuntimeLayoutGeneration, "generation N-1: layout-aware, the compat view is still shipped")
}

func TestInstallManifestConstructorsStampTheBinaryGeneration(t *testing.T) {
	hashes := map[string]string{}
	assert.Equal(t, domain.RuntimeLayoutGeneration, domain.NewInstallManifest("pkg", hashes).RuntimeLayoutGeneration)
	assert.Equal(t, domain.RuntimeLayoutGeneration, domain.NewFullInstallManifest("pkg", hashes).RuntimeLayoutGeneration)
}

// A manifest written by a binary that predates the marker has no field; it must
// read as generation 0 (legacy), never as an error.
func TestInstallManifestWithoutTheFieldReadsAsGenerationZero(t *testing.T) {
	var manifest domain.InstallManifest
	require.NoError(t, json.Unmarshal([]byte(`{"schema":"strategist.install-manifest.v1","package_id":"old","files":[]}`), &manifest))

	assert.Equal(t, 0, manifest.RuntimeLayoutGeneration)
}

func TestInstallManifestGenerationRoundTripsAndIsOmittedWhenZero(t *testing.T) {
	raw, err := json.Marshal(domain.InstallManifest{Schema: "s", RuntimeLayoutGeneration: 2})
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"runtime_layout_generation":2`)

	var back domain.InstallManifest
	require.NoError(t, json.Unmarshal(raw, &back))
	assert.Equal(t, 2, back.RuntimeLayoutGeneration)

	raw, err = json.Marshal(domain.InstallManifest{Schema: "s"})
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "runtime_layout_generation")
}
