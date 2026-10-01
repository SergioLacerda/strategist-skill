package domain

import "fmt"

// CustomPackageEvidence is what the installed providers/<instance>/ files
// say about a Custom package: the facts reconciliation compares to the binding.
type CustomPackageEvidence struct {
	PackageID     string
	Version       string
	PackageDigest string
	AdapterDigest string
	Roles         []string
	Slots         []string
	Entrypoints   []string
}

// ReconcileCustomPackageBinding verifies that the installed package evidence
// agrees with the durable Custom binding, after the shared completeness
// validation. It never repairs or rewrites the lock.
func ReconcileCustomPackageBinding(lock PluginLockFile, role, slot string, evidence CustomPackageEvidence) error {
	binding, err := SingleLockBindingForSlot(lock, slot)
	if err != nil {
		return customRejection("custom binding missing: %v", err)
	}
	if err := ValidateCustomBinding(lock, binding, role, slot); err != nil {
		return err
	}
	if want := CustomInstanceID(evidence.PackageID, evidence.Version); binding.InstalledInstanceID != want {
		return customRejection("slot %q binds %q, installed package evidence identifies %q", slot, binding.InstalledInstanceID, want)
	}
	if evidence.PackageDigest != binding.SourceDigest {
		return customRejection("package digest mismatch for %q: binding=%s installed=%s", evidence.PackageID, binding.SourceDigest, evidence.PackageDigest)
	}
	if evidence.AdapterDigest != binding.WeaponDigest {
		return customRejection("adapter digest mismatch for %q: binding=%s installed=%s", evidence.PackageID, binding.WeaponDigest, evidence.AdapterDigest)
	}
	return reconcileCustomAffinity(binding, role, slot, evidence)
}

func reconcileCustomAffinity(binding SlotBinding, role, slot string, evidence CustomPackageEvidence) error {
	if !containsString(evidence.Roles, role) {
		return customRejection("installed package %q does not declare Role %q", evidence.PackageID, role)
	}
	if !containsString(evidence.Slots, slot) {
		return customRejection("installed package %q does not declare slot %q", evidence.PackageID, slot)
	}
	if !containsString(evidence.Entrypoints, binding.Entrypoint) {
		return customRejection("installed package %q does not declare entrypoint %q", evidence.PackageID, binding.Entrypoint)
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
