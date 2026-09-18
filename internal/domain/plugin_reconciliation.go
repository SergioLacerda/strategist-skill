package domain

import "fmt"

// ReconcileCustomPackageBinding verifies that package evidence agrees with
// the durable Custom binding. It never repairs or rewrites the lock.
func ReconcileCustomPackageBinding(lock PluginLockFile, role, slot string, contract SkillPackageContract) error {
	if err := contract.Validate(); err != nil {
		return fmt.Errorf("plugin reconciliation: invalid package contract: %w", err)
	}
	binding, err := SingleLockBindingForSlot(lock, slot)
	if err != nil {
		return fmt.Errorf("plugin reconciliation: custom binding missing: %w", err)
	}
	if binding.EffectiveMode() != SlotBindingModeCustom {
		return fmt.Errorf("plugin reconciliation: slot %q is %s, not custom", slot, binding.EffectiveMode())
	}
	if binding.InstalledInstanceID != contract.ID {
		return fmt.Errorf("plugin reconciliation: slot %q binds %q, package evidence identifies %q", slot, binding.InstalledInstanceID, contract.ID)
	}
	if got := lock.NodeDigest(contract.ID, string(PluginResourceAdapter)); got == "" {
		return fmt.Errorf("plugin reconciliation: adapter digest missing for %q", contract.ID)
	} else if got != contract.Provenance.NormalizedDigest {
		return fmt.Errorf("plugin reconciliation: adapter digest mismatch for %q: lock=%s package=%s", contract.ID, got, contract.Provenance.NormalizedDigest)
	}
	if got := lock.NodeDigest(role+":"+contract.ID, string(PluginResourceBinding)); got == "" {
		return fmt.Errorf("plugin reconciliation: role binding digest missing for %q", role+":"+contract.ID)
	}
	return nil
}

// ReconcileRankedPackageCertification validates the authoritative Ranked
// catalog stamp. A runtime plugins.lock mirror is intentionally not an input.
func ReconcileRankedPackageCertification(role string, contract SkillPackageContract, stamp CatalogRankedStamp) error {
	if err := contract.Validate(); err != nil {
		return fmt.Errorf("plugin reconciliation: invalid package contract: %w", err)
	}
	if !stamp.Certified() {
		return fmt.Errorf("plugin reconciliation: ranked provider %q is not certified", stamp.ID)
	}
	if stamp.ID != contract.ID {
		return fmt.Errorf("plugin reconciliation: ranked catalog provider %q does not match package %q", stamp.ID, contract.ID)
	}
	if !stamp.HasRole(role) {
		return fmt.Errorf("plugin reconciliation: ranked provider %q does not declare role affinity for %q", stamp.ID, role)
	}
	return nil
}
