package install

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// attachWeaponBindingArtifacts records an evidence projection alongside the
// lock. SlotBinding remains the sole persisted binding authority; the
// projection exists for inspection, onboarding, and taxonomy-aware consumers.
func attachWeaponBindingArtifacts(lock domain.PluginLockFile) (domain.PluginLockFile, error) {
	if len(lock.Bindings) == 0 {
		lock.BindingArtifacts = nil
		return lock, nil
	}
	artifacts, err := domain.NewWeaponBindingArtifacts(lock.Bindings)
	if err != nil {
		return domain.PluginLockFile{}, fmt.Errorf("build Weapon Binding Artifacts: %w", err)
	}
	lock.BindingArtifacts = artifacts
	return lock, nil
}
