package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubAdapter struct{ identity Identity }

func (s stubAdapter) Identity() Identity { return s.identity }
func (s stubAdapter) Call(context.Context, Request) (Result, error) {
	return Result{}, nil
}

func stub(provider, version string, caps ...Capability) stubAdapter {
	return stubAdapter{Identity{Provider: provider, Version: version, Contract: ContractVersion, Capabilities: caps}}
}

func binding(provider, version string, capability Capability) Binding {
	return Binding{Provider: provider, Version: version, Consumer: "handoff", Capability: capability, Model: "jev-1.13.0"}
}

func TestRegistryResolvesOnlyByExplicitBinding(t *testing.T) {
	registry, err := NewRegistry(stub("jev", "1", CapHandoffValidate), stub("other", "1", CapHandoffValidate), stub("jev", "2", CapHandoffValidate))
	require.NoError(t, err)

	adapter, err := registry.Resolve(binding("jev", "1", CapHandoffValidate))
	require.NoError(t, err)
	require.Equal(t, "1", adapter.Identity().Version, "an exact version is returned, never the latest or first registered")

	for name, bad := range map[string]Binding{
		"unknown provider": binding("ghost", "1", CapHandoffValidate),
		"unknown version":  binding("jev", "9", CapHandoffValidate),
		"empty version":    binding("jev", "", CapHandoffValidate),
		"empty provider":   binding("", "1", CapHandoffValidate),
		"latest is a name": binding("jev", "latest", CapHandoffValidate),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := registry.Resolve(bad)
			state, ok := StateOf(err)
			require.True(t, ok)
			require.Equal(t, StateBindingIntegrity, state)
		})
	}
}

func TestRegistryRejectsUnsupportedCapabilityAndContract(t *testing.T) {
	registry, err := NewRegistry(stub("jev", "1", CapHandoffValidate))
	require.NoError(t, err)

	_, err = registry.Resolve(binding("jev", "1", CapHandoffProject))
	state, _ := StateOf(err)
	require.Equal(t, StateUnsupportedCapability, state)

	old := stubAdapter{Identity{Provider: "jev", Version: "3", Contract: "strategist.integration/v0", Capabilities: []Capability{CapHandoffValidate}}}
	registry, err = NewRegistry(old)
	require.NoError(t, err)
	_, err = registry.Resolve(binding("jev", "3", CapHandoffValidate))
	state, _ = StateOf(err)
	require.Equal(t, StateIncompatibleContract, state)
}

func TestRegistryRejectsDuplicateIdentity(t *testing.T) {
	_, err := NewRegistry(stub("jev", "1"), stub("jev", "1"))
	require.Error(t, err)
}

func TestBindingDigestChangesWithEveryIdentityField(t *testing.T) {
	base := binding("jev", "1", CapHandoffValidate)
	seen := map[string]string{base.Digest(): "base"}
	for name, mutate := range map[string]func(*Binding){
		"provider":   func(b *Binding) { b.Provider = "x" },
		"version":    func(b *Binding) { b.Version = "2" },
		"model":      func(b *Binding) { b.Model = "jev-1.14.0" },
		"consumer":   func(b *Binding) { b.Consumer = "other" },
		"capability": func(b *Binding) { b.Capability = CapConfidenceEvaluate },
	} {
		changed := base
		mutate(&changed)
		require.NotContains(t, seen, changed.Digest(), name)
		seen[changed.Digest()] = name
	}
	require.Equal(t, base.Digest(), binding("jev", "1", CapHandoffValidate).Digest(), "deterministic")
}
