package integration

import (
	"fmt"
	"slices"
)

// Registry is the compile-time set of approved adapters. It resolves only by
// explicit provider and version: order, heuristics and "latest" never choose.
type Registry struct {
	adapters map[string]Adapter
}

func adapterKey(provider, version string) string { return provider + "@" + version }

// NewRegistry registers adapters and rejects a duplicate identity.
func NewRegistry(adapters ...Adapter) (*Registry, error) {
	registry := &Registry{adapters: map[string]Adapter{}}
	for _, adapter := range adapters {
		identity := adapter.Identity()
		key := adapterKey(identity.Provider, identity.Version)
		if _, exists := registry.adapters[key]; exists {
			return nil, fmt.Errorf("integration adapter %s is registered twice", key)
		}
		registry.adapters[key] = adapter
	}
	return registry, nil
}

// Resolve returns the adapter an explicit binding names, after checking the
// contract version and the capability.
func (r *Registry) Resolve(binding Binding) (Adapter, error) {
	correlation := Correlation{Provider: binding.Provider, Consumer: binding.Consumer}
	if binding.Provider == "" || binding.Version == "" || binding.Version == "latest" {
		return nil, NewError(StateBindingIntegrity, correlation, "binding must name a provider and an exact version")
	}
	adapter, ok := r.adapters[adapterKey(binding.Provider, binding.Version)]
	if !ok {
		return nil, NewError(StateBindingIntegrity, correlation, "adapter is not in the approved registry")
	}
	identity := adapter.Identity()
	if identity.Contract != ContractVersion {
		return nil, NewError(StateIncompatibleContract, correlation, "adapter implements another contract version")
	}
	if !slices.Contains(identity.Capabilities, binding.Capability) {
		return nil, NewError(StateUnsupportedCapability, correlation, "adapter does not support the capability")
	}
	return adapter, nil
}
