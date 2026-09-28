package check

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"gopkg.in/yaml.v3"
)

// resolveFromCustomBinding is the second step of slot resolution (DEC-010): a
// package added with `strategist provider add` is a custom binding in plugins.lock
// whose staged providers/<instance>/adapter.yaml describes the Weapon. It runs
// after the catalog and before the transitional view, and hands over (handled
// false) whenever no such package is staged, so a legacy binding or a hand-made
// view keeps resolving exactly as before.
func resolveFromCustomBinding(root, slot, provider string) (slotResolution, string, bool) {
	binding, ok := customBindingFor(readPluginsLockFile(root), slot, provider)
	if !ok {
		return slotResolution{}, "", false
	}
	instance := binding.InstalledInstanceID
	facts, found, err := domain.ResolveCustomPackageFacts(root, instance)
	if err != nil {
		return slotResolution{}, fmt.Sprintf("slot %s: custom package %q unreadable: %v", slot, instance, err), true
	}
	if !found {
		return slotResolution{}, "", false
	}
	if instance != provider {
		return slotResolution{}, fmt.Sprintf("slot %s: custom_package_use_instance_id: active.yaml names package %q; name the installed instance %q", slot, provider, instance), true
	}
	if errMsg := checkSlotFacts(root, slot, provider, facts); errMsg != "" {
		return slotResolution{}, errMsg, true
	}
	adapterPath := filepath.Join(root, "providers", instance, "adapter.yaml")
	return slotResolution{kind: slotResolutionSkillProvider, path: adapterPath, readiness: customPackageReadiness(root, slot, instance, adapterPath, facts)}, "", true
}

// customBindingFor finds the custom binding of slot for provider, spelled as the
// installed instance id or as the bare package id.
func customBindingFor(lock domain.PluginLockFile, slot, provider string) (domain.SlotBinding, bool) {
	for _, binding := range lock.Bindings {
		id := binding.InstalledInstanceID
		if binding.Slot == slot && binding.EffectiveMode() == domain.SlotBindingModeCustom && (id == provider || strings.HasPrefix(id, provider+"@")) {
			return binding, true
		}
	}
	return domain.SlotBinding{}, false
}

func customPackageReadiness(root, slot, instance, adapterPath string, facts domain.WeaponFacts) domain.PluginReadinessVector {
	return weaponReadiness(root, slot, instance, readinessFacets{
		descriptor: domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "adapter_contract_valid", Detail: adapterPath},
		source:     domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "custom_package_present", Detail: filepath.Dir(adapterPath)},
		entrypoint: customEntrypointCheck(facts),
		detail:     adapterPath,
		conformance: func(probe connectors.ConnectorResult) domain.ReadinessCheck {
			return customConformanceReadinessFor(root, slot, instance, func() ([]string, domain.ReadinessCheck) {
				if len(facts.Roles) == 0 {
					return nil, conformanceCheck(domain.ReadinessUnknown, "conformance_role_affinity_unknown", "adapter does not declare supported_roles")
				}
				return facts.Roles, domain.ReadinessCheck{}
			}, probe)
		},
		trustAndGrant: customTrustAndGrant,
	})
}

// customEntrypointCheck is a presence check on the adapter's declared entrypoints;
// it does not validate their vocabulary.
func customEntrypointCheck(facts domain.WeaponFacts) domain.ReadinessCheck {
	if len(facts.Entrypoints) == 0 {
		return domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "adapter_entrypoints_missing"}
	}
	return domain.ReadinessCheck{Status: domain.ReadinessReady, ReasonCode: "adapter_entrypoints_declared", Detail: strings.Join(facts.Entrypoints, ",")}
}

// customTrustAndGrant evaluates trust and the permission grant for a custom
// binding. The lock keys its nodes by the package id, not the instance id, so the
// digest is looked up through customPackageID; an unknown package or digest is
// Blocked, never Unknown: a package must not pass with its permissions unevaluated.
func customTrustAndGrant(root, instance string, lock domain.PluginLockFile) (domain.ReadinessCheck, domain.ReadinessCheck) {
	packageID := customPackageID(root, instance)
	digest := ""
	if packageID != "" {
		digest = lock.NodeDigest(packageID, string(domain.PluginResourceAdapter))
	}
	if digest == "" {
		blocked := domain.ReadinessCheck{Status: domain.ReadinessBlocked, ReasonCode: "custom_package_digest_missing", Detail: "no adapter_contract digest in plugins.lock for package " + packageID}
		return blocked, blocked
	}
	return skillProviderTrustReadiness(root, packageID, digest), skillProviderPermissionGrantReadinessFor(root, digest, requestedPermissions(root, instance))
}

// customPackageID is the identity function from an installed instance id to the
// package id the lock keys its nodes by: the id in the staged package.yaml. It
// returns "" when the package manifest is missing or unreadable.
func customPackageID(root, instance string) string {
	raw, err := os.ReadFile(filepath.Join(root, "providers", instance, "package.yaml")) //nolint:gosec // G304: path derived from the runtime root and a locked instance id
	if err != nil {
		return ""
	}
	var pkg struct {
		ID string `yaml:"id"`
	}
	if yaml.Unmarshal(raw, &pkg) != nil {
		return ""
	}
	return pkg.ID
}
