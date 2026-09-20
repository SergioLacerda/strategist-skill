package install

import (
	"fmt"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"gopkg.in/yaml.v3"
)

const pluginCatalogPath = "plugins/catalog.yaml"

type pluginCatalog struct {
	SchemaVersion string                  `yaml:"schema_version"`
	Providers     []pluginCatalogProvider `yaml:"providers"`
}

type pluginCatalogProvider struct {
	ID            string   `yaml:"id"`
	Version       string   `yaml:"version,omitempty"`
	SchemaVersion string   `yaml:"provider_schema_version,omitempty"`
	Status        string   `yaml:"status,omitempty"`
	RiskScore     string   `yaml:"risk_score"`
	Category      string   `yaml:"category,omitempty"`
	CanonicalRole string   `yaml:"canonical_role,omitempty"`
	Roles         []string `yaml:"roles,omitempty"`
	Lifecycle     bool     `yaml:"lifecycle,omitempty"`
	// Default marks this provider as the primary Arma for its CanonicalRole
	// among candidates sharing it — a selection preference, not proof of
	// provenance, installation, or readiness (see domain.ProviderContract.Default).
	Default bool `yaml:"default,omitempty"`
	// Ranked and CertificationDigest mark a build-time-certified Ranked
	// binding candidate — independent of Default (see
	// domain.ProviderContract.Ranked;
	// docs/adr/0043-ranked-pipeline-pilot-implementation-decisions.md DEC-001).
	Ranked              bool   `yaml:"ranked,omitempty"`
	CertificationDigest string `yaml:"certification_digest,omitempty"`
	// RankedBindingGeneration and RankedBindingStatus are the pre-generated
	// runtime SlotBinding fragment stamped by `strategist plugin
	// prepare-embedded`'s certification pass (ADR-0043 DEC-005), copied — not
	// recomputed — by the Wizard's Ranked activation path.
	RankedBindingGeneration int64  `yaml:"ranked_binding_generation,omitempty"`
	RankedBindingStatus     string `yaml:"ranked_binding_status,omitempty"`
	// HostAPIDigest, ConnectorDigest, TestSuiteDigest, PolicyDigest, and ConformanceLevel
	// are ADR-0043 DEC-006's generic Ranked-certification evidence,
	// computed once per (role, provider) pairing by
	// internal/install/embedded_skill_conformance.go — never hardcoded to
	// one pairing. Empty for every non-Ranked entry.
	HostAPIDigest    string                       `yaml:"host_api_digest,omitempty"`
	ConnectorDigest  string                       `yaml:"connector_digest,omitempty"`
	TestSuiteDigest  string                       `yaml:"test_suite_digest,omitempty"`
	PolicyDigest     string                       `yaml:"policy_digest,omitempty"`
	ConformanceLevel string                       `yaml:"conformance_level,omitempty"`
	Runtime          domain.RankedRuntimeContract `yaml:"runtime,omitempty"`
	// UpstreamRepo through License are ADR-0029 DEC-002's per-provider
	// upstream-identity fields — see externalSkillAdapter's own doc comment
	// for the full rationale. Populated only for packages whose upstream
	// provenance has actually been researched.
	UpstreamRepo          string                    `yaml:"upstream_repo,omitempty"`
	UpstreamSkillPath     string                    `yaml:"upstream_skill_path,omitempty"`
	UpstreamVersion       string                    `yaml:"upstream_version,omitempty"`
	UpstreamCommit        string                    `yaml:"upstream_commit,omitempty"`
	UpstreamContentDigest string                    `yaml:"upstream_content_digest,omitempty"`
	License               string                    `yaml:"license,omitempty"`
	Description           string                    `yaml:"description,omitempty"`
	AuxiliaryTools        []string                  `yaml:"auxiliary_tools_allowed,omitempty"`
	Installable           bool                      `yaml:"installable,omitempty"`
	LegacyManifestPath    string                    `yaml:"legacy_manifest_path,omitempty"`
	CompatibilitySource   string                    `yaml:"compatibility_source,omitempty"`
	Dependencies          []pluginCatalogDependency `yaml:"dependencies,omitempty"`
	// SupportedHandoffSchemas declares which RoleContract.HandoffSchema
	// value(s) this weapon's real output conforms to (see
	// domain.ProviderContract.SupportedHandoffSchemas). Omitted/empty means
	// none — the weapon has no declared compatibility with any role's
	// handoff contract on this dimension.
	SupportedHandoffSchemas []string `yaml:"supported_handoff_schemas,omitempty"`
	// ScratchRoot declares whether this weapon creates its own working/scratch
	// files and, if so, that they belong in the runtime domain — see
	// externalSkillAdapter.ScratchRoot. Legal values: "runtime", "none", or
	// omitted (behaves as "none").
	ScratchRoot    string         `yaml:"scratch_root,omitempty"`
	WeaponContract WeaponContract `yaml:"weapon_contract,omitempty"`
}

type pluginCatalogDependency struct {
	ID         string `yaml:"id"`
	Kind       string `yaml:"kind"`
	Constraint string `yaml:"constraint"`
	Optional   bool   `yaml:"optional,omitempty"`
	Reason     string `yaml:"reason,omitempty"`
}

func loadPluginCatalog(extractor domain.FileExtractor) (pluginCatalog, error) {
	data, err := extractor.ReadFile(pluginCatalogPath)
	if err != nil {
		return pluginCatalog{}, fmt.Errorf("read plugin catalog: %w", err)
	}
	return parseCatalogBytes(data)
}

// parseCatalogBytes parses and validates raw catalog.yaml content, shared by
// loadPluginCatalog (embed.FS-backed, runtime path) and PrepareEmbedded
// (plain-filesystem-backed, maintainer/CI path — cmd/strategist's `strategist
// plugin prepare-embedded` operates on real files, not the compiled-in
// embed.FS, since it generates the very files that FS embeds).
func parseCatalogBytes(data []byte) (pluginCatalog, error) {
	var catalog pluginCatalog
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return pluginCatalog{}, fmt.Errorf("plugin catalog: %w", err)
	}
	if catalog.SchemaVersion == "" {
		return pluginCatalog{}, fmt.Errorf("plugin catalog: schema_version is required")
	}
	if len(catalog.Providers) == 0 {
		return pluginCatalog{}, fmt.Errorf("plugin catalog: providers must have at least one entry")
	}
	for _, provider := range catalog.Providers {
		if provider.ID == "" || provider.RiskScore == "" {
			return pluginCatalog{}, fmt.Errorf("plugin catalog: provider id and risk_score are required")
		}
	}
	return catalog, nil
}

func catalogKnownProviderRisk(catalog pluginCatalog) map[string]string {
	providers := make(map[string]string, len(catalog.Providers))
	for _, provider := range catalog.Providers {
		providers[provider.ID] = provider.RiskScore
	}
	return providers
}

// catalogInstallableDefaultProviders and resolveInstallableDefaultProviders
// live in plugin_catalog_resolve.go, split out to keep this file under the
// repo's file-size budget. generateKnownProvidersYAML, catalogResolverCandidates,
// catalogProviderDigest, catalogDependencies, and providerVersionOrDefault
// live in plugin_catalog_digest.go, for the same reason.

func findCatalogProvider(catalog pluginCatalog, providerID string) (pluginCatalogProvider, bool) {
	for _, provider := range catalog.Providers {
		if provider.ID == providerID {
			return provider, true
		}
	}
	return pluginCatalogProvider{}, false
}
