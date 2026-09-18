package install

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
)

// externalSkillAdapter (the strategist.yaml sidecar type) and its
// load/validate/normalize helpers live in embedded_skill_adapter.go, split
// out to keep this file under the repo's file-size budget.

// IngestedSkill is one externally-sourced package that resolved, verified,
// and validated successfully.
type IngestedSkill struct {
	ID               string
	Dir              string
	Package          domain.PluginPackage
	Adapter          externalSkillAdapter
	NormalizedDigest string
}

// IngestionRejection names one candidate directory that did not make it into
// the catalog, and why — ingestion never silently drops a candidate.
type IngestionRejection struct {
	ID     string
	Reason string
}

// IngestionResult is the outcome of one IngestExternalSkills run.
type IngestionResult struct {
	Ingested []IngestedSkill
	Rejected []IngestionRejection
	// Catalog is the existing catalog's providers plus every accepted
	// ingested skill, ready to be written back via WriteCatalogAndMirrors.
	Catalog pluginCatalog
}

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

// resolveExternalSkill resolves one candidate directory into an IngestedSkill
// — the ORKA package via connectors.ResolveLocalPackage, plus its required
// strategist.yaml adapter sidecar.
func resolveExternalSkill(dir string) (IngestedSkill, error) {
	pkg, err := connectors.ResolveLocalPackage(dir)
	if err != nil {
		return IngestedSkill{}, fmt.Errorf("resolve external skill %s: %w", dir, err)
	}
	adapter, err := loadExternalSkillAdapter(dir, pkg.ID)
	if err != nil {
		return IngestedSkill{}, err
	}
	if err := validateIngestedSkillContract(pkg, adapter); err != nil {
		return IngestedSkill{}, fmt.Errorf("external skill %s: %w", pkg.ID, err)
	}
	return IngestedSkill{ID: pkg.ID, Dir: dir, Package: pkg, Adapter: adapter}, nil
}

func validateIngestedSkillContract(pkg domain.PluginPackage, adapter externalSkillAdapter) error {
	roles := append([]string(nil), adapter.Roles...)
	slots := make([]string, 0, len(roles))
	for _, role := range roles {
		switch role {
		case "ranger":
			slots = append(slots, "discovery")
		case "archivist":
			slots = append(slots, "refinement")
		case "sniper":
			slots = append(slots, "execution")
		case "auxiliary":
			// Auxiliary tools are catalogued for explicit dependency
			// resolution, but are never eligible for a mission slot binding.
			slots = append(slots, "auxiliary")
		}
	}
	if adapter.Lifecycle && len(slots) == 0 {
		slots = []string{"discovery", "refinement", "execution"}
	}
	capabilities := append([]string(nil), adapter.Capabilities...)
	if len(capabilities) == 0 {
		capabilities = []string{"role." + adapter.CanonicalRole}
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
func IngestExternalSkills(sourceDir string, existingCatalog pluginCatalog, trustPolicy domain.TrustPolicy) (IngestionResult, error) {
	dirs, err := ScanExternalSkillsSourceDirs(sourceDir)
	if err != nil {
		return IngestionResult{}, err
	}

	var result IngestionResult
	candidates, candidateIDs := resolveCandidates(dirs, &result)
	baseProviders := supersedableBaseProviders(existingCatalog.Providers, candidateIDs)
	accepted := filterAcceptedCandidates(candidates, baseProviders, trustPolicy, &result)
	result.Ingested = filterDependencyResolved(accepted, existingCatalog.SchemaVersion, baseProviders, &result)

	sort.Slice(result.Rejected, func(i, j int) bool { return result.Rejected[i].ID < result.Rejected[j].ID })
	sort.Slice(result.Ingested, func(i, j int) bool { return result.Ingested[i].ID < result.Ingested[j].ID })
	result.Catalog = buildCatalog(existingCatalog.SchemaVersion, baseProviders, result.Ingested)
	return result, nil
}

// resolveCandidates, supersedableBaseProviders, filterAcceptedCandidates,
// filterDependencyResolved, buildCatalog, and trustReasonCodes live in
// embedded_skill_ingestion_filters.go, split out to keep this file under the
// repo's file-size budget.

// catalogProviderFromIngestedSkill and skillDescription live in
// embedded_skill_catalog_entry.go, split out for the same reason.
