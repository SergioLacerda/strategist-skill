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
// This map is only reachable from resolveInstallableDefaultProviders' own
// post-catalog manifest-writing fallback (writeSelectedProviderManifest,
// called after the Wizard has already completed against a successfully
// loaded plugins/catalog.yaml) — see mission
// 20260913-wizard-hardcoded-fallback-maps-cleanup and
// docs/adr/0035-embedded-weapon-fallback-policy.md. It is a permanent,
// intentional default roster (DEC-001), not scheduled for deletion.
// runWizard's own prompt-time flow no longer depends on it: a
// plugins/catalog.yaml load failure now hard-blocks the wizard before any
// prompt is shown (see runWizard), rather than silently degrading to this
// map's values.
var installableDefaultProviders = map[string]string{
	defaultSkillBySlot["discovery"]:  "skills/brainstorming/skill.yaml",
	"openspec-explore":               "skills/openspec-explore/skill.yaml",
	defaultSkillBySlot["refinement"]: "skills/openspec-propose/skill.yaml",
}

// knownProviderRisk is the static fallback for loadKnownProviders' own
// templates/known-providers.yaml tier (a secondary, smaller embedded file,
// independent of plugins/catalog.yaml). It is unreachable from runWizard in
// practice: runWizard hard-blocks before calling loadKnownProviders if
// plugins/catalog.yaml itself fails to load, and loadKnownProviders always
// prefers a successfully loaded catalog first. It remains directly exercised
// by TestLoadKnownProviders and is kept as a defensive fallback for that
// narrower, still-possible failure (see docs/adr/0035-embedded-weapon-fallback-policy.md).
var knownProviderRisk = map[string]string{
	defaultSkillBySlot["discovery"]:  "write_analysis",
	"openspec-explore":               "write_analysis",
	defaultSkillBySlot["refinement"]: "write_analysis",
	"openspec-apply-change":          "controlled",
	"openspec-archive-change":        "write_analysis",
	nativeExecutionProvider:          "controlled",
	"sdd-ask":                        "controlled",
	"batata":                         "controlled",
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
