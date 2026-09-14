package install

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"gopkg.in/yaml.v3"
)

// externalSkillAdapterFileName is the Strategist-owned project adapter
// sidecar living alongside each external-skills-source/<id>/SKILL.md — the
// two-layer contract ADR-0033 defines: SKILL.md stays a portable ORKA
// package (name/description/metadata only); strategist.yaml carries the
// project-specific fields (canonical_role, risk_score, category) SKILL.md
// must never declare itself.
const externalSkillAdapterFileName = "strategist.yaml"

// externalSkillAdapter is the minimal project adapter this mission needs —
// exactly the fields internal/install/plugin_catalog.go#pluginCatalogProvider
// already requires, not a speculative superset. ADR-0033 defers the full
// adapter schema (T2); this is the smallest slice that lets a real ingestion
// pipeline exist today without inventing that schema prematurely.
type externalSkillAdapter struct {
	CanonicalRole  string   `yaml:"canonical_role"`
	RiskScore      string   `yaml:"risk_score"`
	Category       string   `yaml:"category"`
	Default        bool     `yaml:"default,omitempty"`
	AuxiliaryTools []string `yaml:"auxiliary_tools_allowed,omitempty"`
	// SupportedHandoffSchemas — see domain.ProviderContract's field of the
	// same name and internal/install/role_handoff_schemas.go. Omitted by
	// every embedded weapon today — added here only so a future honest
	// declaration isn't silently dropped by ingestion.
	SupportedHandoffSchemas []string `yaml:"supported_handoff_schemas,omitempty"`
}

// IngestedSkill is one externally-sourced package that resolved, verified,
// and validated successfully.
type IngestedSkill struct {
	ID      string
	Dir     string
	Package domain.PluginPackage
	Adapter externalSkillAdapter
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
	adapterPath := filepath.Join(dir, externalSkillAdapterFileName)
	adapterRaw, err := os.ReadFile(adapterPath) //nolint:gosec // G304: dir is an operator-declared ingestion source, not untrusted request input
	if err != nil {
		return IngestedSkill{}, fmt.Errorf("external skill %s: read %s: %w", pkg.ID, externalSkillAdapterFileName, err)
	}
	var adapter externalSkillAdapter
	if err := yaml.Unmarshal(adapterRaw, &adapter); err != nil {
		return IngestedSkill{}, fmt.Errorf("external skill %s: parse %s: %w", pkg.ID, externalSkillAdapterFileName, err)
	}
	if adapter.CanonicalRole == "" || adapter.RiskScore == "" {
		return IngestedSkill{}, fmt.Errorf("external skill %s: %s must declare canonical_role and risk_score", pkg.ID, externalSkillAdapterFileName)
	}
	return IngestedSkill{ID: pkg.ID, Dir: dir, Package: pkg, Adapter: adapter}, nil
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
