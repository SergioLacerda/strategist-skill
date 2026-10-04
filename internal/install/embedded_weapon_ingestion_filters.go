package install

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/trust"
)

// resolveCandidates resolves every scanned directory into an IngestedSkill,
// recording a rejection (never a silent drop) for any that fails to resolve.
func resolveCandidates(dirs []string, result *IngestionResult) (candidates []IngestedSkill, candidateIdentities map[string]bool) {
	candidateIdentities = make(map[string]bool, len(dirs))
	for _, dir := range dirs {
		skill, err := resolveExternalSkill(dir)
		if err != nil {
			result.Rejected = append(result.Rejected, IngestionRejection{ID: filepath.Base(dir), Reason: err.Error()})
			continue
		}
		identity := skillIdentity(skill)
		if candidateIdentities[identity] {
			result.Rejected = append(result.Rejected, IngestionRejection{
				ID:     skill.ID,
				Reason: fmt.Sprintf("duplicate_identity: %s is provided by more than one package directory (%s)", identity, filepath.Base(dir)),
			})
			continue
		}
		candidates = append(candidates, skill)
		candidateIdentities[identity] = true
	}
	return candidates, candidateIdentities
}

// supersedableBaseProviders returns existingProviders minus any embedded
// entry sharing an id@version with this run's candidates — a candidate
// re-ingesting an identity it already holds in the catalog is a refresh, not a
// collision, while another version of the same id is kept alongside it. This
// is what makes IngestExternalSkills idempotent across repeated runs
// (tasks.md Task 2.5): a hand-authored native_role/external entry sharing an
// id with a candidate is still a real collision and is still rejected by
// filterAcceptedCandidates's id_shadowing check.
func supersedableBaseProviders(existingProviders []pluginCatalogProvider, candidateIdentities map[string]bool) []pluginCatalogProvider {
	base := make([]pluginCatalogProvider, 0, len(existingProviders))
	for _, provider := range existingProviders {
		if provider.CompatibilitySource == "embedded" && candidateIdentities[providerIdentity(provider)] {
			continue // superseded by this run's re-ingestion of the same id@version
		}
		base = append(base, provider)
	}
	return base
}

// filterAcceptedCandidates applies id_shadowing and trust verification, in
// that order, recording a rejection for each candidate that fails either.
func filterAcceptedCandidates(candidates []IngestedSkill, baseProviders []pluginCatalogProvider, trustPolicy domain.TrustPolicy, result *IngestionResult) []IngestedSkill {
	existingIDs := shadowingIDs(baseProviders)

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
	violationsByID := dependencyViolations(buildCatalog(schemaVersion, baseProviders, accepted), accepted)
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

// dependencyViolations maps each provider with an unresolved dependency, or a
// composite whose composition is invalid, to what is missing.
func dependencyViolations(prospective pluginCatalog, accepted []IngestedSkill) map[string][]string {
	violationsByID := map[string][]string{}
	for _, violation := range ValidateCatalogDependencies(prospective) {
		violationsByID[violation.ProviderID] = append(violationsByID[violation.ProviderID], violation.DependencyID)
	}
	if err := validateCatalogWeaponCompositions(prospective); err != nil {
		addCompositionViolations(violationsByID, accepted, err)
	}
	return violationsByID
}

func addCompositionViolations(violationsByID map[string][]string, accepted []IngestedSkill, err error) {
	for _, skill := range accepted {
		if skill.Adapter.Kind == "composite" {
			violationsByID[skill.ID] = append(violationsByID[skill.ID], err.Error())
		}
	}
}

// buildCatalog merges baseProviders with skills, sorted by id.
func buildCatalog(schemaVersion string, baseProviders []pluginCatalogProvider, skills []IngestedSkill) pluginCatalog {
	catalog := pluginCatalog{SchemaVersion: schemaVersion, TaxonomyVersion: domain.CanonicalTaxonomyVersion, Providers: append([]pluginCatalogProvider(nil), baseProviders...)}
	for _, skill := range skills {
		catalog.Providers = append(catalog.Providers, catalogProviderFromIngestedSkill(skill))
	}
	sortCatalogProviders(catalog.Providers)
	return catalog
}

func trustReasonCodes(result trust.Result) []string {
	codes := make([]string, 0, len(result.Reasons))
	for _, reason := range result.Reasons {
		codes = append(codes, reason.Code)
	}
	return codes
}

// shadowingIDs returns the ids a candidate may not reuse. Only a non-embedded
// entry shadows an id, whatever its version: an embedded entry of another
// version of the same id coexists (ADR-0061 Decision 8).
func shadowingIDs(baseProviders []pluginCatalogProvider) map[string]bool {
	ids := make(map[string]bool, len(baseProviders))
	for _, provider := range baseProviders {
		if provider.CompatibilitySource != "embedded" {
			ids[provider.ID] = true
		}
	}
	return ids
}
