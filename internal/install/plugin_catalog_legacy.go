package install

import (
	"bytes"
	"fmt"
	"strings"
)

func generateLegacyProviderManifest(catalog pluginCatalog, providerID string) ([]byte, error) {
	provider, ok := findCatalogProvider(catalog, providerID)
	if !ok {
		return nil, fmt.Errorf("plugin catalog: provider %q not found", providerID)
	}
	var buf bytes.Buffer
	writeLegacyProviderField(&buf, "id", provider.ID)
	writeLegacyProviderQuotedField(&buf, "version", provider.Version)
	writeLegacyProviderQuotedField(&buf, "schema_version", provider.SchemaVersion)
	writeLegacyProviderField(&buf, "status", provider.Status)
	writeLegacyProviderField(&buf, "risk_score", provider.RiskScore)
	writeLegacyProviderField(&buf, "category", provider.Category)
	writeLegacyProviderField(&buf, "provider_class", provider.ProviderClass)
	buf.WriteString("\n")
	buf.WriteString("description: >\n")
	for _, line := range strings.Split(provider.Description, "\n") {
		if strings.TrimSpace(line) == "" {
			buf.WriteString("\n")
			continue
		}
		buf.WriteString("  " + line + "\n")
	}
	buf.WriteString("\n")
	buf.WriteString("specialization_taxonomy:\n")
	writeLegacyProviderIndentedField(&buf, "canonical_role", provider.CanonicalRole)
	writeLegacyProviderIndentedField(&buf, "provider_class", provider.ProviderClass)
	if len(provider.AuxiliaryTools) > 0 {
		buf.WriteString("\n")
		buf.WriteString("auxiliary_tools_allowed:\n")
		for _, tool := range provider.AuxiliaryTools {
			buf.WriteString("  - " + tool + "\n")
		}
	}
	return buf.Bytes(), nil
}

func writeLegacyProviderField(buf *bytes.Buffer, key, value string) {
	fmt.Fprintf(buf, "%s: %s\n", key, value)
}

func writeLegacyProviderQuotedField(buf *bytes.Buffer, key, value string) {
	fmt.Fprintf(buf, "%s: %q\n", key, value)
}

func writeLegacyProviderIndentedField(buf *bytes.Buffer, key, value string) {
	fmt.Fprintf(buf, "  %s: %s\n", key, value)
}
