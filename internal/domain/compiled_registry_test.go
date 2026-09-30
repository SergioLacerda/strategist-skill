package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validCompiledRegistry() domain.CompiledRegistry {
	return domain.CompiledRegistry{
		SchemaVersion: domain.CompiledRegistrySchemaVersion,
		Weapons: []domain.CompiledWeapon{{
			ID: "brainstorming", Version: "1.0.0", Digest: "sha256:weapon", Origin: domain.WeaponOriginEmbedded,
			SourceDigest: "sha256:source", Runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded, ExecutionMode: domain.WeaponExecutionModePromptBridge},
		}},
		Roles: []domain.CompiledRole{{ID: "ranger", Slot: "discovery", ContractDigest: "sha256:role"}},
		RankedBindings: []domain.CompiledRankedBinding{{
			Role: "ranger", Slot: "discovery", WeaponID: "brainstorming", WeaponVersion: "1.0.0", WeaponDigest: "sha256:weapon", RoleDigest: "sha256:role", BindingDigest: "sha256:binding", CertificationDigest: "sha256:cert",
			SourceDigest: "sha256:source", ExecutionMode: domain.WeaponExecutionModePromptBridge, ConnectorID: "strategist-embedded", Runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded, ExecutionMode: domain.WeaponExecutionModePromptBridge}, Entrypoint: "discover", Generation: 1, Status: "active",
		}},
	}
}

func TestCompiledRegistryValidateAndLookup(t *testing.T) {
	t.Parallel()

	registry := validCompiledRegistry()
	require.NoError(t, registry.Validate())

	weapon, ok := registry.Weapon("brainstorming", "1.0.0")
	assert.True(t, ok)
	assert.Equal(t, domain.WeaponOriginEmbedded, weapon.Origin)

	role, ok := registry.Role("ranger")
	assert.True(t, ok)
	assert.Equal(t, "discovery", role.Slot)

	binding, ok := registry.RankedBinding("ranger", "discovery", "brainstorming", "1.0.0")
	assert.True(t, ok)
	assert.Equal(t, "brainstorming", binding.WeaponID)
}

func TestCompiledRegistryRejectsDuplicateAndBrokenBindings(t *testing.T) {
	t.Parallel()

	t.Run("duplicate role", func(t *testing.T) {
		registry := validCompiledRegistry()
		registry.Roles = append(registry.Roles, registry.Roles[0])
		require.ErrorContains(t, registry.Validate(), "duplicate Role")
	})

	t.Run("unknown weapon", func(t *testing.T) {
		registry := validCompiledRegistry()
		registry.RankedBindings[0].WeaponID = "missing"
		require.ErrorContains(t, registry.Validate(), "unknown Weapon")
	})

	t.Run("embedded host api", func(t *testing.T) {
		registry := validCompiledRegistry()
		registry.RankedBindings[0].Runtime.HostAPI = "host/v1"
		require.Error(t, registry.Validate())
	})
}

func TestCompiledRegistryWeaponIdentityIsIDAndVersion(t *testing.T) {
	t.Parallel()

	second := func() domain.CompiledRegistry {
		registry := validCompiledRegistry()
		newer := registry.Weapons[0]
		newer.Version = "2.0.0"
		newer.Digest = "sha256:weapon-2"
		registry.Weapons = append(registry.Weapons, newer)
		return registry
	}

	t.Run("two versions of one Weapon coexist", func(t *testing.T) {
		registry := second()
		require.NoError(t, registry.Validate())
		old, ok := registry.Weapon("brainstorming", "1.0.0")
		require.True(t, ok)
		assert.Equal(t, "sha256:weapon", old.Digest)
		newer, ok := registry.Weapon("brainstorming", "2.0.0")
		require.True(t, ok)
		assert.Equal(t, "sha256:weapon-2", newer.Digest)
	})

	t.Run("lookup without a known version does not resolve", func(t *testing.T) {
		registry := second()
		_, ok := registry.Weapon("brainstorming", "")
		assert.False(t, ok, "an omitted version must never resolve to some version")
		_, ok = registry.Weapon("brainstorming", "9.9.9")
		assert.False(t, ok)
	})

	t.Run("same id and version twice is a duplicate", func(t *testing.T) {
		registry := validCompiledRegistry()
		registry.Weapons = append(registry.Weapons, registry.Weapons[0])
		require.ErrorContains(t, registry.Validate(), "duplicate Weapon \"brainstorming@1.0.0\"")
	})

	t.Run("weapon without a version is rejected", func(t *testing.T) {
		registry := validCompiledRegistry()
		registry.Weapons[0].Version = ""
		require.ErrorContains(t, registry.Validate(), "version is required")
	})

	t.Run("ranked binding pins one version and its digest", func(t *testing.T) {
		registry := second()
		registry.RankedBindings[0].WeaponVersion = "2.0.0" // digest still names 1.0.0
		require.ErrorContains(t, registry.Validate(), "digest mismatch")
		registry.RankedBindings[0].WeaponDigest = "sha256:weapon-2"
		require.NoError(t, registry.Validate())
	})

	t.Run("ranked binding without a version is rejected", func(t *testing.T) {
		registry := validCompiledRegistry()
		registry.RankedBindings[0].WeaponVersion = ""
		require.ErrorContains(t, registry.Validate(), "version is required")
	})

	t.Run("ranked binding to an unknown version is rejected", func(t *testing.T) {
		registry := validCompiledRegistry()
		registry.RankedBindings[0].WeaponVersion = "9.9.9"
		require.ErrorContains(t, registry.Validate(), "unknown Weapon")
	})
}

func twoVersionRegistry() domain.CompiledRegistry {
	registry := validCompiledRegistry()
	newer := registry.Weapons[0]
	newer.Version, newer.Digest = "2.0.0", "sha256:weapon-2"
	registry.Weapons = append(registry.Weapons, newer)
	binding := registry.RankedBindings[0]
	binding.WeaponVersion, binding.WeaponDigest, binding.BindingDigest, binding.CertificationDigest = "2.0.0", "sha256:weapon-2", "sha256:binding-2", "sha256:cert-2"
	registry.RankedBindings = append(registry.RankedBindings, binding)
	return registry
}

func TestCompiledRegistryCertifiesOneRankedBindingPerVersion(t *testing.T) {
	t.Parallel()

	registry := twoVersionRegistry()
	require.NoError(t, registry.Validate(), "two certified versions may serve the same role and slot")

	old, ok := registry.RankedBinding("ranger", "discovery", "brainstorming", "1.0.0")
	require.True(t, ok)
	assert.Equal(t, "sha256:binding", old.BindingDigest)
	newer, ok := registry.RankedBinding("ranger", "discovery", "brainstorming", "2.0.0")
	require.True(t, ok)
	assert.Equal(t, "sha256:binding-2", newer.BindingDigest)

	_, ok = registry.RankedBinding("ranger", "discovery", "brainstorming", "")
	assert.False(t, ok, "a role and slot alone never pick a version")
	_, ok = registry.RankedBinding("ranger", "discovery", "brainstorming", "3.0.0")
	assert.False(t, ok)

	offers := registry.RankedBindingsFor("ranger", "discovery")
	require.Len(t, offers, 2)
	assert.Equal(t, []string{"1.0.0", "2.0.0"}, []string{offers[0].WeaponVersion, offers[1].WeaponVersion}, "sorted by version")
	assert.Empty(t, registry.RankedBindingsFor("archivist", "refinement"))
}

func TestCompiledRegistryRejectsTheSameRankedBindingTwice(t *testing.T) {
	t.Parallel()

	registry := validCompiledRegistry()
	registry.RankedBindings = append(registry.RankedBindings, registry.RankedBindings[0])
	require.ErrorContains(t, registry.Validate(), "duplicate Ranked binding ranger/discovery/brainstorming@1.0.0")
}

func TestParseWeaponRef(t *testing.T) {
	t.Parallel()

	id, version := domain.ParseWeaponRef("brainstorming@2.0.0")
	assert.Equal(t, "brainstorming", id)
	assert.Equal(t, "2.0.0", version)

	id, version = domain.ParseWeaponRef("brainstorming")
	assert.Equal(t, "brainstorming", id)
	assert.Empty(t, version, "a plain id carries no version")

	id, version = domain.ParseWeaponRef("brainstorming@")
	assert.Equal(t, "brainstorming", id)
	assert.Empty(t, version)
}
