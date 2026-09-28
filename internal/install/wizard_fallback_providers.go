package install

import (
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

// installableDefaultProviders lists providers that ship as an installable
// skill.yaml template. archivist is deliberately absent here: it is the
// native refinement role (roles/archivist.yaml), materialized like any other
// native role, not a skill package requiring its own install manifest.
//
// This map is retained as build/test metadata for embedded provider manifests;
// it is not a runtime role/weapon fallback. See mission
// 20260913-wizard-hardcoded-fallback-maps-cleanup and
// docs/adr/0035-embedded-weapon-fallback-policy.md. It is a permanent,
// intentional default roster (DEC-001), not scheduled for deletion.
// runWizard's own prompt-time flow no longer depends on it: a
// plugins/catalog.yaml load failure now hard-blocks the wizard before any
// prompt is shown (see runWizard), rather than silently degrading to this
// map's values.
var installableDefaultProviders = map[string]string{
	defaultSkillBySlot["discovery"]:  "brainstorming",
	"openspec-explore":               "openspec-explore",
	defaultSkillBySlot["refinement"]: "openspec-propose",
}

// knownProviderRisk is historical static metadata for loadKnownProviders'
// templates/known-providers.yaml tier (a secondary, smaller embedded file,
// independent of plugins/catalog.yaml). It is unreachable from runWizard in
// practice: runWizard hard-blocks before calling loadKnownProviders if
// plugins/catalog.yaml itself fails to load, and loadKnownProviders always
// prefers a successfully loaded catalog first. It remains directly exercised
// by TestLoadKnownProviders; it is outside the active Wizard/check readiness
// path.
var knownProviderRisk = map[string]string{
	defaultSkillBySlot["discovery"]:  "write_analysis",
	"openspec-explore":               "write_analysis",
	defaultSkillBySlot["refinement"]: "write_analysis",
	"openspec-apply-change":          "controlled",
	"openspec-archive-change":        "controlled",
	nativeExecutionProvider:          "controlled",
	"sdd-ask":                        "controlled",
	"sdd-diagnose":                   "write_analysis",
	"sdd-converge":                   "controlled",
	"sdd-correct":                    "controlled",
	"sdd-stabilize":                  "controlled",
	"sdd-validate-governance":        "write_analysis",
	"sdd-organize":                   "write_analysis",
	"sdd-review-architecture":        "write_analysis",
	"archivist":                      "write_analysis",
}

// loadKnownProviders reads templates/known-providers.yaml from the extractor and
// returns a provider→risk_score map. Falls back to the static map on any error.
func loadKnownProviders(extractor domain.FileExtractor) map[string]string {
	if catalog, err := loadPluginCatalog(extractor); err == nil {
		return catalogKnownProviderRisk(catalog)
	}
	data, err := extractor.ReadFile(knownProvidersTemplatePath)
	if err != nil {
		return knownProviderRisk
	}
	var doc struct {
		Providers map[string]string `yaml:"providers"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil || len(doc.Providers) == 0 {
		return knownProviderRisk
	}
	return doc.Providers
}
