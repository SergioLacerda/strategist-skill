package domain

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// CustomRoleBindingNodeKind is the PluginLockNode.Kind that records one
// resolved Role→Weapon binding. internal/plugins.RoleBindingLockKind holds the
// same value for the Ranked/migration producers.
const CustomRoleBindingNodeKind = "role_provider_binding"

// CustomReinstallGuidance is appended to every Custom binding rejection: an
// incomplete or inconsistent binding is never repaired or inferred, only
// replaced by running the corrected onboarding flow again.
const CustomReinstallGuidance = "re-add the package with `strategist provider add` or reinstall it through the wizard"

// CustomPackageFacts are the normalized, installed facts a Custom binding is
// derived from. Package and adapter lock nodes are keyed by the bare PackageID;
// the binding and its role-binding node are keyed by the versioned instance id.
type CustomPackageFacts struct {
	PackageID      string
	PackageVersion string
	Role           string
	Slot           string
	PackageDigest  string
	AdapterDigest  string
	RuntimeKind    string
	ConnectorID    string
	Entrypoint     string
}

// CustomInstanceID is the deterministic versioned instance identity,
// "<package id>@<version>". It is never interchangeable with the bare id.
func CustomInstanceID(packageID, version string) string {
	return packageID + "@" + version
}

// InstanceID returns the versioned instance id of the package.
func (f CustomPackageFacts) InstanceID() string {
	return CustomInstanceID(f.PackageID, f.PackageVersion)
}

// CustomBindingEvidence is the complete binding plus the lock nodes that back
// it. Callers publish both together or neither.
type CustomBindingEvidence struct {
	Binding SlotBinding
	Nodes   []PluginLockNode
}

// NewCustomBindingEvidence derives the complete Custom SlotBinding and its
// package, adapter and role-binding lock nodes from normalized package facts.
// Incomplete or inconsistent facts are rejected before anything is published.
func NewCustomBindingEvidence(facts CustomPackageFacts, generation int64, status string) (CustomBindingEvidence, error) {
	if err := facts.validate(); err != nil {
		return CustomBindingEvidence{}, customRejection("%v", err)
	}
	instance := facts.InstanceID()
	digest := customRoleBindingDigest(facts)
	binding := SlotBinding{
		SchemaVersion: "strategist-plugin-binding/v1", TaxonomyVersion: CanonicalTaxonomyVersion, Slot: facts.Slot, InstalledInstanceID: instance, Role: facts.Role,
		WeaponVersion: facts.PackageVersion, WeaponDigest: facts.AdapterDigest, SourceDigest: facts.PackageDigest,
		BindingDigest: digest, Origin: string(WeaponOriginCustom), RuntimeKind: facts.RuntimeKind,
		ConnectorID: facts.ConnectorID, Entrypoint: facts.Entrypoint,
		Generation: generation, Status: status, Mode: SlotBindingModeCustom,
	}
	nodes := []PluginLockNode{
		{ID: facts.PackageID, Kind: string(PluginResourcePackage), Digest: facts.PackageDigest},
		{ID: facts.PackageID, Kind: string(PluginResourceAdapter), Digest: facts.AdapterDigest},
		{ID: facts.Role + ":" + instance, Kind: CustomRoleBindingNodeKind, Digest: digest},
	}
	evidence := CustomBindingEvidence{Binding: binding, Nodes: nodes}
	return evidence, nil
}

func (f CustomPackageFacts) validate() error {
	required := []struct{ name, value string }{
		{"package id", f.PackageID}, {"package version", f.PackageVersion}, {"role", f.Role}, {"slot", f.Slot},
		{"package digest", f.PackageDigest}, {"adapter digest", f.AdapterDigest},
		{"runtime kind", f.RuntimeKind}, {"connector", f.ConnectorID}, {"entrypoint", f.Entrypoint},
	}
	for _, field := range required {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("custom package facts are missing %s", field.name)
		}
	}
	if strings.Contains(f.PackageID, "@") {
		return fmt.Errorf("custom package id %q must be the bare id; the versioned instance id is derived", f.PackageID)
	}
	if role, ok := DefaultRoleRegistry().RoleForSlot(f.Slot); !ok || role.ID != f.Role {
		return fmt.Errorf("custom package role %q does not own slot %q", f.Role, f.Slot)
	}
	return nil
}

// customRoleBindingDigest commits the binding to every identity field, so a
// tampered binding no longer matches the digest recorded in the lock.
func customRoleBindingDigest(f CustomPackageFacts) string {
	fields := []string{"custom-role-binding/v1", f.Role, f.Slot, f.InstanceID(), f.PackageDigest, f.AdapterDigest, f.RuntimeKind, f.ConnectorID, f.Entrypoint}
	sum := sha256.Sum256([]byte(strings.Join(fields, "\t")))
	return fmt.Sprintf("sha256:%x", sum)
}

func customRejection(format string, args ...any) error {
	return fmt.Errorf("custom_binding_invalid: "+format+"; "+CustomReinstallGuidance, args...)
}

// ValidateCustomBinding is the one pure, fail-closed validator every Custom
// consumer shares (binding resolution, the lock-plan constructor, check and
// mission planning). It never repairs, infers or rewrites anything: a missing,
// mismatched or tampered value is an error that names the evidence.
func ValidateCustomBinding(lock PluginLockFile, binding SlotBinding, role, slot string) error {
	if err := ValidateTaxonomyVersion(binding.TaxonomyVersion); err != nil {
		return customRejection("slot %q has invalid taxonomy version: %v", slot, err)
	}
	if err := validateCustomBindingFields(binding, role, slot); err != nil {
		return err
	}
	packageID, _ := ParseWeaponRef(binding.InstalledInstanceID)
	facts := CustomPackageFacts{
		PackageID: packageID, PackageVersion: binding.WeaponVersion, Role: role, Slot: slot,
		PackageDigest: binding.SourceDigest, AdapterDigest: binding.WeaponDigest,
		RuntimeKind: binding.RuntimeKind, ConnectorID: binding.ConnectorID, Entrypoint: binding.Entrypoint,
	}
	if binding.BindingDigest != customRoleBindingDigest(facts) {
		return customRejection("slot %q binding digest does not match its recorded identity (tampered or inconsistent binding)", slot)
	}
	return validateCustomLockEvidence(lock, binding, facts)
}

func validateCustomBindingFields(binding SlotBinding, role, slot string) error {
	if binding.EffectiveMode() != SlotBindingModeCustom {
		return customRejection("slot %q is %s, not custom", slot, binding.EffectiveMode())
	}
	if err := requireCustomBindingFields(binding, slot); err != nil {
		return err
	}
	if binding.Slot != slot || binding.Generation < 1 {
		return customRejection("slot %q binding is for slot %q at generation %d", slot, binding.Slot, binding.Generation)
	}
	if binding.Role != role {
		return customRejection("slot %q binding belongs to Role %q, not %q", slot, binding.Role, role)
	}
	id, version := ParseWeaponRef(binding.InstalledInstanceID)
	if id == "" || version == "" || version != binding.WeaponVersion {
		return customRejection("slot %q instance %q is not the versioned %q instance id", slot, binding.InstalledInstanceID, "id@"+binding.WeaponVersion)
	}
	return nil
}

// requireCustomBindingFields names the first missing piece of identity.
func requireCustomBindingFields(binding SlotBinding, slot string) error {
	required := []struct{ name, value string }{
		{"role", binding.Role}, {"weapon version", binding.WeaponVersion}, {"package or Weapon digest", binding.WeaponDigest},
		{"source digest", binding.SourceDigest}, {"role-binding digest", binding.BindingDigest}, {"origin", binding.Origin},
		{"runtime kind", binding.RuntimeKind}, {"connector", binding.ConnectorID}, {"entrypoint", binding.Entrypoint}, {"status", binding.Status},
	}
	for _, field := range required {
		if strings.TrimSpace(field.value) == "" {
			return customRejection("slot %q Custom binding has no %s", slot, field.name)
		}
	}
	return nil
}

func validateCustomLockEvidence(lock PluginLockFile, binding SlotBinding, facts CustomPackageFacts) error {
	checks := []struct {
		name, id, kind, want string
	}{
		{"package", facts.PackageID, string(PluginResourcePackage), binding.SourceDigest},
		{"adapter", facts.PackageID, string(PluginResourceAdapter), binding.WeaponDigest},
		{"role binding", facts.Role + ":" + binding.InstalledInstanceID, CustomRoleBindingNodeKind, binding.BindingDigest},
	}
	for _, check := range checks {
		got := lock.NodeDigest(check.id, check.kind)
		if got == "" {
			return customRejection("plugins.lock has no %s evidence for %q", check.name, check.id)
		}
		if got != check.want {
			return customRejection("%s digest for %q differs between the binding and plugins.lock", check.name, check.id)
		}
	}
	return nil
}
