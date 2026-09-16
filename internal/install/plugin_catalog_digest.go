package install

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/plugins"
)

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
