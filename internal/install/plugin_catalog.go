package install

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins"
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
	// Default marks this provider as the primary Arma for its CanonicalRole
	// among candidates sharing it — a selection preference, not proof of
	// provenance, installation, or readiness (see domain.ProviderContract.Default).
	Default             bool                      `yaml:"default,omitempty"`
	Description         string                    `yaml:"description,omitempty"`
	AuxiliaryTools      []string                  `yaml:"auxiliary_tools_allowed,omitempty"`
	Installable         bool                      `yaml:"installable,omitempty"`
	LegacyManifestPath  string                    `yaml:"legacy_manifest_path,omitempty"`
	CompatibilitySource string                    `yaml:"compatibility_source,omitempty"`
	Dependencies        []pluginCatalogDependency `yaml:"dependencies,omitempty"`
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
	ScratchRoot string `yaml:"scratch_root,omitempty"`
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
// repo's file-size budget.

func generateKnownProvidersYAML(catalog pluginCatalog) []byte {
	var buf bytes.Buffer
	buf.WriteString("# Generated from plugins/catalog.yaml. Do not edit by hand.\n")
	buf.WriteString("providers:\n")
	providers := append([]pluginCatalogProvider(nil), catalog.Providers...)
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].ID < providers[j].ID
	})
	for _, provider := range providers {
		fmt.Fprintf(&buf, "  %s: %s\n", provider.ID, provider.RiskScore)
	}
	return buf.Bytes()
}

func catalogResolverCandidates(catalog pluginCatalog) []plugins.Candidate {
	providers := append([]pluginCatalogProvider(nil), catalog.Providers...)
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].ID < providers[j].ID
	})
	candidates := make([]plugins.Candidate, 0, len(providers))
	for _, provider := range providers {
		candidates = append(candidates, plugins.Candidate{
			ID:           provider.ID,
			Kind:         "adapter_contract",
			Version:      providerVersionOrDefault(provider.Version),
			Digest:       catalogProviderDigest(provider),
			Dependencies: catalogDependencies(provider.Dependencies),
		})
	}
	return candidates
}

func catalogProviderDigest(provider pluginCatalogProvider) string {
	if provider.Installable {
		if data, err := generateLegacyProviderManifest(pluginCatalog{SchemaVersion: "digest", Providers: []pluginCatalogProvider{provider}}, provider.ID); err == nil {
			sum := sha256.Sum256(data)
			return fmt.Sprintf("sha256:%x", sum)
		}
	}
	var b strings.Builder
	b.WriteString(provider.ID)
	b.WriteString("\t")
	b.WriteString(providerVersionOrDefault(provider.Version))
	b.WriteString("\t")
	b.WriteString(provider.RiskScore)
	b.WriteString("\t")
	b.WriteString(provider.CompatibilitySource)
	b.WriteString("\n")
	sum := sha256.Sum256([]byte(b.String()))
	return fmt.Sprintf("sha256:%x", sum)
}

func catalogDependencies(dependencies []pluginCatalogDependency) []plugins.Dependency {
	out := make([]plugins.Dependency, 0, len(dependencies))
	for _, dep := range dependencies {
		out = append(out, plugins.Dependency{
			ID:         dep.ID,
			Kind:       dep.Kind,
			Constraint: dep.Constraint,
			Optional:   dep.Optional,
			Reason:     dep.Reason,
		})
	}
	return out
}

func providerVersionOrDefault(version string) string {
	if version == "" {
		return "0.0.0"
	}
	return version
}

func findCatalogProvider(catalog pluginCatalog, providerID string) (pluginCatalogProvider, bool) {
	for _, provider := range catalog.Providers {
		if provider.ID == providerID {
			return provider, true
		}
	}
	return pluginCatalogProvider{}, false
}
