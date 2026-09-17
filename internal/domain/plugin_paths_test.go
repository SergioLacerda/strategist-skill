package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeStrictPluginYAMLRejectsMalformedSyntax(t *testing.T) {
	t.Parallel()

	err := domain.DecodeStrictPluginYAML([]byte("id: [unterminated"), &domain.PluginPackage{})
	require.ErrorContains(t, err, "plugin yaml decode")
}

func TestValidatePluginRelativePathRejectsCurrentDirectory(t *testing.T) {
	t.Parallel()

	err := domain.ValidatePluginRelativePath(".")
	require.ErrorContains(t, err, "escape")
}

func TestPluginPackageValidationRejectsMissingAndMalformedDigest(t *testing.T) {
	t.Parallel()

	base := domain.PluginPackage{
		SchemaVersion:   "strategist-plugin-package/v1",
		ID:              "openai/brainstorming",
		Version:         "1.0.0",
		ArtifactURI:     "embedded://skills/brainstorming",
		ArtifactSize:    1024,
		License:         "MIT",
		CreatedAt:       "2026-08-20T00:00:00Z",
		ManifestSchema:  "strategist-plugin-package/v1",
		UpstreamVersion: "1.0.0",
	}

	base.Digest = ""
	err := base.Validate()
	require.ErrorContains(t, err, "digest is required")

	base.Digest = "not-a-real-digest"
	err = base.Validate()
	require.ErrorContains(t, err, "digest must be sha256:<64 lowercase hex>")
}

func TestAdapterContractCheckCompatibilityPluginAPIEdgeCases(t *testing.T) {
	t.Parallel()

	contract := domain.AdapterContract{PluginAPIRange: ">=1 <2"}
	baseVector := domain.PluginVersionVector{
		ManifestSchema:  "strategist-plugin-package/v1",
		AdapterRevision: "adapter-rev-20260820",
		UpstreamVersion: "1.0.0",
		ConnectorAPI:    "strategist-connector-api/1",
	}

	t.Run("plugin api missing expected prefix", func(t *testing.T) {
		t.Parallel()
		vector := baseVector
		vector.PluginAPI = "bogus"
		result := contract.CheckCompatibility(vector)
		assert.False(t, result.Compatible)
	})

	t.Run("plugin api major is non-numeric", func(t *testing.T) {
		t.Parallel()
		vector := baseVector
		vector.PluginAPI = "strategist-plugin-api/abc"
		result := contract.CheckCompatibility(vector)
		assert.False(t, result.Compatible)
	})

	t.Run("range expression is empty", func(t *testing.T) {
		t.Parallel()
		vector := baseVector
		vector.PluginAPI = "strategist-plugin-api/1"
		emptyRange := domain.AdapterContract{PluginAPIRange: ""}
		result := emptyRange.CheckCompatibility(vector)
		assert.False(t, result.Compatible)
	})

	t.Run("range part has unrecognized operator", func(t *testing.T) {
		t.Parallel()
		vector := baseVector
		vector.PluginAPI = "strategist-plugin-api/1"
		tildeRange := domain.AdapterContract{PluginAPIRange: "~1"}
		result := tildeRange.CheckCompatibility(vector)
		assert.False(t, result.Compatible)
	})

	t.Run("range part is too short to carry an operator", func(t *testing.T) {
		t.Parallel()
		vector := baseVector
		vector.PluginAPI = "strategist-plugin-api/1"
		shortRange := domain.AdapterContract{PluginAPIRange: "="}
		result := shortRange.CheckCompatibility(vector)
		assert.False(t, result.Compatible)
	})

	t.Run("exact match operator accepts equal major", func(t *testing.T) {
		t.Parallel()
		vector := baseVector
		vector.PluginAPI = "strategist-plugin-api/1"
		exactRange := domain.AdapterContract{PluginAPIRange: "=1"}
		result := exactRange.CheckCompatibility(vector)
		assert.True(t, result.Compatible)
	})

	t.Run("exact match operator rejects different major", func(t *testing.T) {
		t.Parallel()
		vector := baseVector
		vector.PluginAPI = "strategist-plugin-api/2"
		exactRange := domain.AdapterContract{PluginAPIRange: "=1"}
		result := exactRange.CheckCompatibility(vector)
		assert.False(t, result.Compatible)
	})
}
