package domain_test

import (
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginResourceVocabularySeparatesAuthorities(t *testing.T) {
	t.Parallel()

	resources := domain.PluginResourceAuthorities()
	require.Len(t, resources, 8)

	seen := map[string]string{}
	for _, resource := range resources {
		require.NotEmpty(t, resource.Kind)
		require.NotEmpty(t, resource.Authority)
		require.NotEmpty(t, resource.Owns)
		for _, field := range resource.Owns {
			if previous := seen[field]; previous != "" {
				t.Fatalf("field %q is owned by both %s and %s", field, previous, resource.Kind)
			}
			seen[field] = string(resource.Kind)
		}
	}

	assert.Equal(t, string(domain.PluginResourcePackage), seen["publisher_identity"])
	assert.Equal(t, string(domain.PluginResourceAdapter), seen["host_api_compatibility"])
	assert.Equal(t, string(domain.PluginResourceInventory), seen["verification_evidence"])
	assert.Equal(t, string(domain.PluginResourceBinding), seen["slot_selection"])
	assert.Equal(t, string(domain.PluginResourceTrustPolicy), seen["trusted_publishers"])
	assert.Equal(t, string(domain.PluginResourceGrant), seen["granted_permissions"])
	assert.Equal(t, string(domain.PluginResourceLock), seen["resolved_graph"])
	assert.Equal(t, string(domain.PluginResourceTransaction), seen["journaled_transition"])
}

func TestPluginVersionVectorRequiresIndependentDimensions(t *testing.T) {
	t.Parallel()

	vector := domain.PluginVersionVector{
		ManifestSchema:  "strategist-plugin-package/v1",
		PluginAPI:       "strategist-plugin-api/1",
		AdapterRevision: "adapter-rev-20260820",
		UpstreamVersion: "1.2.3",
		ConnectorAPI:    "strategist-connector-api/1",
	}
	require.NoError(t, vector.Validate())

	err := domain.PluginVersionVector{ManifestSchema: "strategist-plugin-package/v1"}.Validate()
	require.Error(t, err)
	require.ErrorContains(t, err, "plugin_api is required")
	require.ErrorContains(t, err, "adapter_revision is required")
	require.ErrorContains(t, err, "upstream_version is required")
	require.ErrorContains(t, err, "connector_api is required")
}

func TestPluginPackageValidationRejectsMixedAuthorityAndUnboundedInput(t *testing.T) {
	t.Parallel()

	pkg := domain.PluginPackage{
		SchemaVersion:   "strategist-plugin-package/v1",
		ID:              "openai/brainstorming",
		Version:         "1.0.0",
		Digest:          "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ArtifactURI:     "embedded://skills/brainstorming",
		ArtifactSize:    1024,
		License:         "MIT",
		CreatedAt:       "2026-08-20T00:00:00Z",
		ManifestSchema:  "strategist-plugin-package/v1",
		UpstreamVersion: "1.0.0",
	}
	require.NoError(t, pkg.Validate())

	pkg.BindingStatus = "active"
	err := pkg.Validate()
	require.Error(t, err)
	require.ErrorContains(t, err, "binding_status belongs to SlotBinding")

	pkg.BindingStatus = ""
	pkg.ArtifactSize = domain.MaxPluginManifestBytes + 1
	err = pkg.Validate()
	require.Error(t, err)
	assert.ErrorContains(t, err, "artifact_size exceeds")
}

func TestAdapterContractValidationReturnsStructuredCompatibility(t *testing.T) {
	t.Parallel()

	contract := domain.AdapterContract{
		SchemaVersion:     "strategist-plugin-adapter/v1",
		ID:                "strategist/brainstorming-adapter",
		AdapterRevision:   "adapter-rev-20260820",
		PluginAPIRange:    ">=1 <2",
		SupportedSlots:    []string{"discovery"},
		Entrypoints:       []string{"discover"},
		PackageConstraint: "openai/brainstorming@>=1",
		RequestedPermissions: []domain.PluginPermission{
			domain.PluginPermissionReadWorkspace,
		},
	}
	require.NoError(t, contract.Validate())

	result := contract.CheckCompatibility(domain.PluginVersionVector{
		ManifestSchema:  "strategist-plugin-package/v1",
		PluginAPI:       "strategist-plugin-api/1",
		AdapterRevision: "adapter-rev-20260820",
		UpstreamVersion: "1.0.0",
		ConnectorAPI:    "strategist-connector-api/1",
	})
	assert.True(t, result.Compatible)
	assert.Empty(t, result.Reasons)

	result = contract.CheckCompatibility(domain.PluginVersionVector{
		ManifestSchema:  "strategist-plugin-package/v1",
		PluginAPI:       "strategist-plugin-api/3",
		AdapterRevision: "adapter-rev-20260820",
		UpstreamVersion: "1.0.0",
		ConnectorAPI:    "strategist-connector-api/1",
	})
	assert.False(t, result.Compatible)
	assert.Contains(t, result.Reasons, domain.CompatibilityReason{
		Dimension: "plugin_api",
		Code:      "unsupported_host_api",
		Detail:    "strategist-plugin-api/3 is outside >=1 <2",
	})
}

func TestDecodeStrictPluginYAMLRejectsUnknownFieldsAndOversize(t *testing.T) {
	t.Parallel()

	valid := []byte(strings.Join([]string{
		"schema_version: strategist-plugin-package/v1",
		"id: openai/brainstorming",
		"version: 1.0.0",
		"digest: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"artifact_uri: embedded://skills/brainstorming",
		"artifact_size: 1024",
		"license: MIT",
		"created_at: 2026-08-20T00:00:00Z",
		"manifest_schema: strategist-plugin-package/v1",
		"upstream_version: 1.0.0",
	}, "\n"))

	var pkg domain.PluginPackage
	require.NoError(t, domain.DecodeStrictPluginYAML(valid, &pkg))
	require.NoError(t, pkg.Validate())

	withUnknown := append([]byte{}, valid...)
	withUnknown = append(withUnknown, []byte("\nlocal_health: ready\n")...)
	err := domain.DecodeStrictPluginYAML(withUnknown, &domain.PluginPackage{})
	require.Error(t, err)
	require.ErrorContains(t, err, "unknown field")

	oversized := make([]byte, domain.MaxPluginManifestBytes+1)
	err = domain.DecodeStrictPluginYAML(oversized, &domain.PluginPackage{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "exceeds")
}

func TestValidatePluginRelativePathBounds(t *testing.T) {
	t.Parallel()

	require.NoError(t, domain.ValidatePluginRelativePath("packages/openai/brainstorming/package.yaml"))

	for _, path := range []string{
		"",
		"/absolute/package.yaml",
		"..\\escape\\package.yaml",
		"packages/../escape/package.yaml",
		strings.Repeat("a", domain.MaxPluginPathLength+1),
	} {
		err := domain.ValidatePluginRelativePath(path)
		require.Error(t, err, "path=%q", path)
	}
}
