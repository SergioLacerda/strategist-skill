package domain

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// factsFromAdapter resolves a package added with `strategist provider add`: the
// plugins.lock must hold a binding with mode custom whose instance id is the
// provider (or provider@version) and the staged providers/<instance>/adapter.yaml
// must exist. A staged package that is not bound as custom is not a resolved Weapon.
func factsFromAdapter(strategistRoot, provider string) (WeaponFacts, bool, error) {
	instance, err := customInstance(strategistRoot, provider)
	if err != nil || instance == "" {
		return WeaponFacts{}, false, err
	}
	raw, err := os.ReadFile(filepath.Join(strategistRoot, "providers", instance, "adapter.yaml")) //nolint:gosec // G304: path derived from the runtime root and a locked instance id
	if errors.Is(err, os.ErrNotExist) {
		return WeaponFacts{}, false, nil
	}
	if err != nil {
		return WeaponFacts{}, false, fmt.Errorf("read adapter for %s: %w", instance, err)
	}
	var adapter AdapterContract
	if err := yaml.Unmarshal(raw, &adapter); err != nil {
		return WeaponFacts{}, false, fmt.Errorf("parse adapter for %s: %w", instance, err)
	}
	return adapterFacts(provider, adapter), true, nil
}

func adapterFacts(provider string, adapter AdapterContract) WeaponFacts {
	facts := WeaponFacts{
		ID: provider, Source: WeaponFactsSourceAdapter, RiskScore: adapter.RiskScore, Roles: adapter.SupportedRoles,
		ScratchRoot: adapter.ScratchRoot, SupportedSlots: adapter.SupportedSlots, RequestedPermissions: adapter.RequestedPermissions,
	}
	if len(adapter.SupportedRoles) > 0 {
		facts.CanonicalRole = adapter.SupportedRoles[0]
	}
	return facts
}

// customInstance returns the installed instance id bound with mode custom for
// provider, or "" when the lock has none.
func customInstance(strategistRoot, provider string) (string, error) {
	lock, err := readPluginsLock(strategistRoot)
	if err != nil {
		return "", err
	}
	for _, binding := range lock.Bindings {
		if isCustomBindingFor(binding, provider) {
			return binding.InstalledInstanceID, nil
		}
	}
	return "", nil
}

func isCustomBindingFor(binding SlotBinding, provider string) bool {
	id := binding.InstalledInstanceID
	return binding.EffectiveMode() == SlotBindingModeCustom && (id == provider || strings.HasPrefix(id, provider+"@"))
}

func readPluginsLock(strategistRoot string) (PluginLockFile, error) {
	raw, err := os.ReadFile(filepath.Join(strategistRoot, "plugins.lock")) //nolint:gosec // G304: fixed path under the runtime root
	if errors.Is(err, os.ErrNotExist) {
		return PluginLockFile{}, nil
	}
	if err != nil {
		return PluginLockFile{}, fmt.Errorf("read plugins.lock: %w", err)
	}
	var lock PluginLockFile
	if err := yaml.Unmarshal(raw, &lock); err != nil {
		return PluginLockFile{}, fmt.Errorf("parse plugins.lock: %w", err)
	}
	return lock, nil
}
