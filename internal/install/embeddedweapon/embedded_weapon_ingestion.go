package embeddedweapon

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
)

// externalSkillAdapter (the strategist.yaml sidecar type) and its
// load/validate/normalize helpers live in embedded_weapon_adapter.go, split
// out to keep this file under the repo's file-size budget.

// ScanExternalSkillsSourceDirs lists the immediate subdirectories of
// sourceDir — each one is a candidate ORKA package (tasks.md Task 1/2).
// Returns an empty, non-error result when sourceDir does not exist, since an
// operator who has not created the directory yet is a valid, non-error state
// (matches request point 1: the directory is optional/operator-declared).
func ScanExternalSkillsSourceDirs(sourceDir string) ([]string, error) {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan external skills source %s: %w", sourceDir, err)
	}
	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(sourceDir, entry.Name()))
		}
	}
	sort.Strings(dirs)
	return dirs, nil
}

// ResolveExternalSkill resolves one candidate directory into an IngestedSkill
// — the ORKA package via connectors.ResolveLocalPackage, plus its required
// strategist.yaml adapter sidecar.
func ResolveExternalSkill(dir string) (plugincatalog.IngestedSkill, error) {
	pkg, err := connectors.ResolveEmbeddedPackage(dir)
	if err != nil {
		return plugincatalog.IngestedSkill{}, fmt.Errorf("resolve external skill %s: %w", dir, err)
	}
	adapter, err := plugincatalog.LoadExternalSkillAdapter(dir, pkg.ID)
	if err != nil {
		return plugincatalog.IngestedSkill{}, err //nolint:wrapcheck // sibling package of the same module; its error already names the adapter file
	}
	if err := VerifyUpstreamPin(dir, pkg.ID, adapter); err != nil {
		return plugincatalog.IngestedSkill{}, err
	}
	if pkg.Version == "" {
		pkg.Version = adapter.Version
	}
	if pkg.Version == "" {
		return plugincatalog.IngestedSkill{}, fmt.Errorf("external skill %s: SKILL.md or %s must declare version", pkg.ID, plugincatalog.ExternalSkillAdapterFileName)
	}
	if err := validateIngestedSkillContract(pkg, adapter); err != nil {
		return plugincatalog.IngestedSkill{}, fmt.Errorf("external skill %s: %w", pkg.ID, err)
	}
	return plugincatalog.IngestedSkill{ID: pkg.ID, Dir: dir, Package: pkg, Adapter: adapter}, nil
}

// VerifyUpstreamPin fails when a declared upstream_content_digest pin does not
// match the local SKILL.md bytes and the adapter does not admit that it is an
// adapted copy. An absent pin is explicitly unverified and is not an error.
func VerifyUpstreamPin(dir, id string, adapter plugincatalog.ExternalSkillAdapter) error {
	if adapter.UpstreamContentDigest == "" || adapter.LocalModifications {
		return nil
	}
	local, err := plugincatalog.HashFileSHA256(filepath.Join(dir, names.SkillMDName))
	if err != nil {
		return fmt.Errorf("external skill %s: %w", id, err)
	}
	if local != adapter.UpstreamContentDigest {
		return fmt.Errorf("external skill %s: upstream_pin_mismatch: SKILL.md hashes to %s but upstream_content_digest pins %s; declare local_modifications: true if the copy is intentionally adapted", id, local, adapter.UpstreamContentDigest)
	}
	return nil
}

func validateIngestedSkillContract(pkg domain.PluginPackage, adapter plugincatalog.ExternalSkillAdapter) error {
	roles := append([]string(nil), adapter.Roles...)
	slots := append([]string(nil), adapter.SupportedSlots...)
	capabilities := append([]string(nil), adapter.Capabilities...)
	if len(capabilities) == 0 {
		capabilities = []string{"role." + roles[0]}
	}
	contract := domain.NewSkillPackageContract(pkg, domain.AdapterContract{
		SupportedRoles: roles, SupportedSlots: slots, SupportedHandoffSchemas: adapter.SupportedHandoffSchemas,
		Capabilities: capabilities, PluginAPIRange: "v1",
	})
	if err := contract.Validate(); err != nil {
		return fmt.Errorf("validate skill package contract: %w", err)
	}
	return nil
}

// IngestExternalSkills scans sourceDir, resolves and verifies every
// candidate package, and merges the accepted ones into existingCatalog —
// tasks.md Task 2 (extends P2 to run over N externally-sourced packages).
//
// Order of checks per candidate, each capable of producing a rejection
// (never a silent drop, per tasks.md Task 2.5):
//  1. package + adapter resolution (malformed SKILL.md, missing/invalid
//     strategist.yaml)
//  2. trust.Verify against trustPolicy (C0 package-static gate)
//  3. id_shadowing against the existing catalog (internal/plugins/role_binding.go's
//     rule, reused verbatim — no second collision policy)
//  4. dependency resolution (install.ValidateCatalogDependencies, called
//     against the full prospective merged catalog so a dependency ingested
//     in the same batch as its master still resolves)
func IngestExternalSkills(sourceDir string, existingCatalog plugincatalog.PluginCatalog, trustPolicy domain.TrustPolicy) (plugincatalog.IngestionResult, error) {
	dirs, err := ScanExternalSkillsSourceDirs(sourceDir)
	if err != nil {
		return plugincatalog.IngestionResult{}, err
	}

	var result plugincatalog.IngestionResult
	candidates, candidateIdentities := resolveCandidates(dirs, &result)
	baseProviders := supersedableBaseProviders(existingCatalog.Providers, candidateIdentities)
	accepted := filterAcceptedCandidates(candidates, baseProviders, trustPolicy, &result)
	result.Ingested = filterDependencyResolved(accepted, existingCatalog.SchemaVersion, baseProviders, &result)

	sort.Slice(result.Rejected, func(i, j int) bool { return result.Rejected[i].ID < result.Rejected[j].ID })
	sort.Slice(result.Ingested, func(i, j int) bool {
		return plugincatalog.SkillIdentity(result.Ingested[i]) < plugincatalog.SkillIdentity(result.Ingested[j])
	})
	result.Catalog = buildCatalog(existingCatalog.SchemaVersion, baseProviders, result.Ingested)
	return result, nil
}

// resolveCandidates, supersedableBaseProviders, filterAcceptedCandidates,
// filterDependencyResolved, buildCatalog, and trustReasonCodes live in
// embedded_weapon_ingestion_filters.go, split out to keep this file under the
// repo's file-size budget.

// catalogProviderFromIngestedSkill and skillDescription live in
// embedded_weapon_catalog_entry.go, split out for the same reason.
