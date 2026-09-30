package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRoleWeaponBinding_RankedRequiresCompiledParity(t *testing.T) {
	t.Parallel()
	registry := validCompiledRegistry()
	active := domain.ActiveConfig{Slots: map[string]string{"discovery": "brainstorming"}}
	lock := domain.PluginLockFile{Bindings: []domain.SlotBinding{{
		Slot: "discovery", Role: "ranger", InstalledInstanceID: "brainstorming", WeaponVersion: "1.0.0", Mode: domain.SlotBindingModeRanked,
		WeaponDigest: "sha256:weapon", SourceDigest: "sha256:source", BindingDigest: "sha256:binding",
		ExecutionMode: domain.WeaponExecutionModePromptBridge,
	}}}

	binding, err := domain.ResolveRoleWeaponBinding(active, lock, registry, "ranger", "discovery")
	require.NoError(t, err)
	assert.Equal(t, domain.SlotBindingModeRanked, binding.Mode)
	assert.Equal(t, "strategist-embedded", binding.ConnectorID)
	assert.Equal(t, "1.0.0", binding.WeaponVersion)
}

func TestResolveRoleWeaponBinding_RankedRequiresTheLockedWeaponVersion(t *testing.T) {
	t.Parallel()
	registry := validCompiledRegistry()
	active := domain.ActiveConfig{Slots: map[string]string{"discovery": "brainstorming"}}
	lockFor := func(version string) domain.PluginLockFile {
		return domain.PluginLockFile{Bindings: []domain.SlotBinding{{
			Slot: "discovery", Role: "ranger", InstalledInstanceID: "brainstorming", WeaponVersion: version, Mode: domain.SlotBindingModeRanked,
			WeaponDigest: "sha256:weapon", SourceDigest: "sha256:source", BindingDigest: "sha256:binding",
			ExecutionMode: domain.WeaponExecutionModePromptBridge,
		}}}
	}

	_, err := domain.ResolveRoleWeaponBinding(active, lockFor(""), registry, "ranger", "discovery")
	require.ErrorContains(t, err, "weapon version")

	_, err = domain.ResolveRoleWeaponBinding(active, lockFor("2.0.0"), registry, "ranger", "discovery")
	require.ErrorContains(t, err, "no compiled Ranked binding for ranger/discovery/brainstorming@2.0.0")
	require.ErrorContains(t, err, "certified: brainstorming@1.0.0")
}

func TestResolveRoleWeaponBinding_RejectsIntentAndLockDrift(t *testing.T) {
	t.Parallel()
	active := domain.ActiveConfig{Slots: map[string]string{"discovery": "brainstorming"}}
	lock := domain.PluginLockFile{Bindings: []domain.SlotBinding{{Slot: "discovery", InstalledInstanceID: "other"}}}

	_, err := domain.ResolveRoleWeaponBinding(active, lock, domain.CompiledRegistry{}, "ranger", "discovery")
	require.ErrorContains(t, err, "does not match")
}

func TestResolveRoleWeaponBinding_CustomUsesPersistedBinding(t *testing.T) {
	t.Parallel()
	active := domain.ActiveConfig{Slots: map[string]string{"discovery": "third-party"}}
	lock := domain.PluginLockFile{Bindings: []domain.SlotBinding{{
		Slot: "discovery", Role: "ranger", InstalledInstanceID: "third-party", Mode: domain.SlotBindingModeCustom,
		Origin: string(domain.WeaponOriginCustom), RuntimeKind: domain.RankedRuntimeHost, ConnectorID: "host", Entrypoint: "discover",
	}}}

	binding, err := domain.ResolveRoleWeaponBinding(active, lock, domain.CompiledRegistry{}, "ranger", "discovery")
	require.NoError(t, err)
	assert.Equal(t, domain.SlotBindingModeCustom, binding.Mode)
	assert.Equal(t, "host", binding.ConnectorID)
}

func TestResolveRoleWeaponBinding_RankedSelectsTheLockedVersionAmongCertifiedOnes(t *testing.T) {
	t.Parallel()
	registry := twoVersionRegistry()
	active := domain.ActiveConfig{Slots: map[string]string{"discovery": "brainstorming"}}
	lock := domain.PluginLockFile{Bindings: []domain.SlotBinding{{
		Slot: "discovery", Role: "ranger", InstalledInstanceID: "brainstorming", WeaponVersion: "2.0.0", Mode: domain.SlotBindingModeRanked,
		WeaponDigest: "sha256:weapon-2", SourceDigest: "sha256:source", BindingDigest: "sha256:binding-2",
		ExecutionMode: domain.WeaponExecutionModePromptBridge,
	}}}

	binding, err := domain.ResolveRoleWeaponBinding(active, lock, registry, "ranger", "discovery")

	require.NoError(t, err)
	assert.Equal(t, "2.0.0", binding.WeaponVersion)
	assert.Equal(t, "sha256:binding-2", binding.BindingDigest)
}
