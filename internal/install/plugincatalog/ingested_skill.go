package plugincatalog

import "github.com/SergioLacerda/strategist-skill/internal/domain"

// IngestedSkill is one externally-sourced package that resolved, verified,
// and validated successfully.
type IngestedSkill struct {
	ID               string
	Dir              string
	Package          domain.PluginPackage
	Adapter          ExternalSkillAdapter
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
	Catalog PluginCatalog
}
