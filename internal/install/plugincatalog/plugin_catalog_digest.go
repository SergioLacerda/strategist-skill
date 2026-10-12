package plugincatalog

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/tools/resolver"
)

// GenerateKnownProvidersYAML renders the provider-to-risk map of catalog as the known-providers YAML document.
func GenerateKnownProvidersYAML(catalog PluginCatalog) []byte {
	var buf bytes.Buffer
	buf.WriteString("# Generated from plugins/catalog.yaml. Do not edit by hand.\n")
	buf.WriteString("providers:\n")
	providers := append([]Provider(nil), catalog.Providers...)
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].ID < providers[j].ID
	})
	for _, provider := range providers {
		fmt.Fprintf(&buf, "  %s: %s\n", provider.ID, provider.RiskScore)
	}
	return buf.Bytes()
}

// CatalogResolverCandidates returns one resolver candidate per catalog provider, ordered by provider id.
func CatalogResolverCandidates(catalog PluginCatalog) []resolver.Candidate {
	providers := append([]Provider(nil), catalog.Providers...)
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].ID < providers[j].ID
	})
	candidates := make([]resolver.Candidate, 0, len(providers))
	for _, provider := range providers {
		candidates = append(candidates, resolver.Candidate{
			ID:           provider.ID,
			Kind:         "adapter_contract",
			Version:      ProviderVersionOrDefault(provider.Version),
			Digest:       CatalogProviderDigest(provider),
			Dependencies: catalogDependencies(provider.Dependencies),
		})
	}
	return candidates
}

// CatalogProviderDigest returns the content digest of one catalog provider entry.
func CatalogProviderDigest(provider Provider) string {
	if provider.PackageDigest != "" {
		return provider.PackageDigest
	}
	if provider.Installable {
		if data, err := NormalizedDigestManifest(PluginCatalog{SchemaVersion: "digest", Providers: []Provider{provider}}, provider.ID); err == nil {
			sum := sha256.Sum256(data)
			return fmt.Sprintf("sha256:%x", sum)
		}
	}
	var b strings.Builder
	b.WriteString(provider.ID)
	b.WriteString("\t")
	b.WriteString(ProviderVersionOrDefault(provider.Version))
	b.WriteString("\t")
	b.WriteString(provider.RiskScore)
	b.WriteString("\t")
	b.WriteString(provider.CompatibilitySource)
	fmt.Fprintf(&b, "\tlifecycle=%t", provider.Lifecycle)
	b.WriteString("\n")
	sum := sha256.Sum256([]byte(b.String()))
	return fmt.Sprintf("sha256:%x", sum)
}

func catalogDependencies(dependencies []Dependency) []resolver.Dependency {
	out := make([]resolver.Dependency, 0, len(dependencies))
	for _, dep := range dependencies {
		out = append(out, resolver.Dependency{
			ID:         dep.ID,
			Kind:       dep.Kind,
			Constraint: dep.Constraint,
			Optional:   dep.Optional,
			Reason:     dep.Reason,
		})
	}
	return out
}

// ProviderVersionOrDefault returns version, or the default version when it is empty.
func ProviderVersionOrDefault(version string) string {
	if version == "" {
		return "0.0.0"
	}
	return version
}
