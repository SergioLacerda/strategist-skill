package install

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/connectors"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/trust"
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

// resolveCandidates resolves every scanned directory into an IngestedSkill,
// recording a rejection (never a silent drop) for any that fails to resolve.
func resolveCandidates(dirs []string, result *IngestionResult) (candidates []IngestedSkill, candidateIDs map[string]bool) {
	candidateIDs = make(map[string]bool, len(dirs))
	for _, dir := range dirs {
		skill, err := resolveExternalSkill(dir)
		if err != nil {
			result.Rejected = append(result.Rejected, IngestionRejection{ID: filepath.Base(dir), Reason: err.Error()})
			continue
		}
		candidates = append(candidates, skill)
		candidateIDs[skill.ID] = true
	}
	return candidates, candidateIDs
}

// supersedableBaseProviders returns existingProviders minus any embedded
// entry sharing an id with this run's candidates — a candidate re-ingesting
// an id it already holds in the catalog is a refresh, not a collision. This
// is what makes IngestExternalSkills idempotent across repeated runs
// (tasks.md Task 2.5): a hand-authored native_role/external entry sharing an
// id with a candidate is still a real collision and is still rejected by
// filterAcceptedCandidates's id_shadowing check.
func supersedableBaseProviders(existingProviders []pluginCatalogProvider, candidateIDs map[string]bool) []pluginCatalogProvider {
	base := make([]pluginCatalogProvider, 0, len(existingProviders))
	for _, provider := range existingProviders {
		if provider.CompatibilitySource == "embedded" && candidateIDs[provider.ID] {
			continue // superseded by this run's re-ingestion of the same id
		}
		base = append(base, provider)
	}
	return base
}

// filterAcceptedCandidates applies id_shadowing and trust verification, in
// that order, recording a rejection for each candidate that fails either.
func filterAcceptedCandidates(candidates []IngestedSkill, baseProviders []pluginCatalogProvider, trustPolicy domain.TrustPolicy, result *IngestionResult) []IngestedSkill {
	existingIDs := make(map[string]bool, len(baseProviders))
	for _, provider := range baseProviders {
		existingIDs[provider.ID] = true
	}

	now := time.Now()
	var accepted []IngestedSkill
	for _, skill := range candidates {
		if existingIDs[skill.ID] {
			result.Rejected = append(result.Rejected, IngestionRejection{
				ID:     skill.ID,
				Reason: fmt.Sprintf("id_shadowing: id=%s collides with a non-embedded catalog entry — remove it first or choose a different id", skill.ID),
			})
			continue
		}
		verification := trust.Verify(trust.Subject{Package: skill.Package}, trustPolicy, now)
		if !verification.Trusted {
			result.Rejected = append(result.Rejected, IngestionRejection{
				ID:     skill.ID,
				Reason: fmt.Sprintf("trust_verification_failed: %s", trustReasonCodes(verification)),
			})
			continue
		}
		accepted = append(accepted, skill)
	}
	return accepted
}

// filterDependencyResolved runs dependency resolution against the full
// prospective catalog (base + every accepted candidate) so a dependency
// ingested in the same batch as its master resolves correctly, recording a
// rejection for any candidate whose declared dependency is still missing.
func filterDependencyResolved(accepted []IngestedSkill, schemaVersion string, baseProviders []pluginCatalogProvider, result *IngestionResult) []IngestedSkill {
	prospective := buildCatalog(schemaVersion, baseProviders, accepted)
	violationsByID := map[string][]string{}
	for _, violation := range ValidateCatalogDependencies(prospective) {
		violationsByID[violation.ProviderID] = append(violationsByID[violation.ProviderID], violation.DependencyID)
	}

	var ingested []IngestedSkill
	for _, skill := range accepted {
		if missing, ok := violationsByID[skill.ID]; ok {
			result.Rejected = append(result.Rejected, IngestionRejection{
				ID:     skill.ID,
				Reason: fmt.Sprintf("dependency_unresolved: missing %v", missing),
			})
			continue
		}
		ingested = append(ingested, skill)
	}
	return ingested
}

// buildCatalog merges baseProviders with skills, sorted by id.
func buildCatalog(schemaVersion string, baseProviders []pluginCatalogProvider, skills []IngestedSkill) pluginCatalog {
	catalog := pluginCatalog{SchemaVersion: schemaVersion, Providers: append([]pluginCatalogProvider(nil), baseProviders...)}
	for _, skill := range skills {
		catalog.Providers = append(catalog.Providers, catalogProviderFromIngestedSkill(skill))
	}
	sort.Slice(catalog.Providers, func(i, j int) bool { return catalog.Providers[i].ID < catalog.Providers[j].ID })
	return catalog
}

func trustReasonCodes(result trust.Result) []string {
	codes := make([]string, 0, len(result.Reasons))
	for _, reason := range result.Reasons {
		codes = append(codes, reason.Code)
	}
	return codes
}

func catalogProviderFromIngestedSkill(skill IngestedSkill) pluginCatalogProvider {
	return pluginCatalogProvider{
		ID:                  skill.ID,
		Version:             skill.Package.Version,
		SchemaVersion:       "1",
		Status:              "active",
		RiskScore:           skill.Adapter.RiskScore,
		Category:            skill.Adapter.Category,
		CanonicalRole:       skill.Adapter.CanonicalRole,
		Default:             skill.Adapter.Default,
		Description:         skillDescription(skill.Dir),
		AuxiliaryTools:      skill.Adapter.AuxiliaryTools,
		Installable:         true,
		LegacyManifestPath:  "skills/" + skill.ID + "/skill.yaml",
		CompatibilitySource: "embedded",
	}
}

// skillDescription reads the ORKA-portable "description" frontmatter field
// straight from SKILL.md for the generated catalog entry — a thin metadata
// mirror only, never vendoring the package's own prose body (ADR-0029
// DEC-001). Returns an empty string if SKILL.md is missing or malformed;
// resolveExternalSkill already rejects that case earlier in the pipeline, so
// this is a defensive fallback, not the primary validation path.
func skillDescription(dir string) string {
	raw, err := os.ReadFile(filepath.Join(dir, "SKILL.md")) //nolint:gosec // G304: dir is an operator-declared ingestion source, not untrusted request input
	if err != nil {
		return ""
	}
	var fm struct {
		Description string `yaml:"description"`
	}
	content := string(raw)
	const delim = "---"
	if len(content) < len(delim) || content[:len(delim)] != delim {
		return ""
	}
	rest := content[len(delim):]
	end := -1
	needle := "\n" + delim
	for i := 0; i+len(needle) <= len(rest); i++ {
		if rest[i:i+len(needle)] == needle {
			end = i
			break
		}
	}
	if end < 0 {
		return ""
	}
	if err := yaml.Unmarshal([]byte(rest[:end]), &fm); err != nil {
		return ""
	}
	return fm.Description
}
