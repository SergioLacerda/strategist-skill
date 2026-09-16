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
	writeLegacyProviderField(&buf, "canonical_role", provider.CanonicalRole)
	writeLegacyRoles(&buf, provider.Roles)
	writeLegacyScratchRoot(&buf, provider.ScratchRoot)
	buf.WriteString("\n")
	writeLegacyDescription(&buf, provider.Description)
	writeLegacyAuxiliaryTools(&buf, provider.AuxiliaryTools)
	return buf.Bytes(), nil
}

func writeLegacyRoles(buf *bytes.Buffer, roles []string) {
	if len(roles) == 0 {
		return
	}
	buf.WriteString("roles:\n")
	for _, role := range roles {
		buf.WriteString("  - " + role + "\n")
	}
}

func writeLegacyScratchRoot(buf *bytes.Buffer, scratchRoot string) {
	if scratchRoot != "" {
		writeLegacyProviderField(buf, "scratch_root", scratchRoot)
	}
}

func writeLegacyDescription(buf *bytes.Buffer, description string) {
	buf.WriteString("description: >\n")
	for _, line := range strings.Split(description, "\n") {
		if strings.TrimSpace(line) == "" {
			buf.WriteString("\n")
			continue
		}
		buf.WriteString("  " + line + "\n")
	}
}

func writeLegacyAuxiliaryTools(buf *bytes.Buffer, tools []string) {
	if len(tools) == 0 {
		return
	}
	buf.WriteString("\nauxiliary_tools_allowed:\n")
	for _, tool := range tools {
		buf.WriteString("  - " + tool + "\n")
	}
}

func writeLegacyProviderField(buf *bytes.Buffer, key, value string) {
	fmt.Fprintf(buf, "%s: %s\n", key, value)
}

func writeLegacyProviderQuotedField(buf *bytes.Buffer, key, value string) {
	fmt.Fprintf(buf, "%s: %q\n", key, value)
}
