package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteWeaponCatalogWritesEntriesAndPayloads(t *testing.T) {
	root := t.TempDir()

	WriteWeaponCatalog(t, root,
		CatalogProvider{ID: "brainstorming", Risk: "write_analysis", CanonicalRole: "ranger"},
		CatalogProvider{ID: "sdd-ask", Risk: "controlled", Source: "external", RuntimeKind: "embedded"},
		CatalogProvider{ID: "sniper", Risk: "controlled", Source: "native_role"},
		CatalogProvider{ID: "no-payload", Risk: "write_analysis", NoPayload: true},
	)

	raw, err := os.ReadFile(filepath.Join(root, "plugins", "catalog.yaml"))
	require.NoError(t, err)
	text := string(raw)
	assert.Contains(t, text, "schema_version: strategist-plugin-catalog/v2")
	assert.Contains(t, text, "  - id: brainstorming\n    risk_score: write_analysis\n    compatibility_source: embedded\n")
	assert.Contains(t, text, "role_owner: ranger", "a ranger Weapon carries its weapon_contract")
	assert.Contains(t, text, "      kind: host", "the default runtime kind")
	assert.Contains(t, text, "      kind: embedded", "an explicit runtime kind")
	assert.Contains(t, text, "  - id: sniper\n    risk_score: controlled\n    compatibility_source: native_role\n")
	assert.FileExists(t, filepath.Join(root, "skills", "brainstorming", "SKILL.md"))
	assert.FileExists(t, filepath.Join(root, "skills", "sdd-ask", "SKILL.md"))
	assert.NoFileExists(t, filepath.Join(root, "skills", "sniper", "SKILL.md"), "a native role has no Weapon payload")
	assert.NoFileExists(t, filepath.Join(root, "skills", "no-payload", "SKILL.md"))
}

func TestCatalogEntryYAMLRendersOneEntry(t *testing.T) {
	entry := CatalogEntryYAML(CatalogProvider{ID: "x", Risk: "write_analysis", CanonicalRole: "archivist"})

	assert.Contains(t, entry, "  - id: x\n")
	assert.Contains(t, entry, "    canonical_role: archivist\n    roles:\n      - archivist\n")
	assert.NotContains(t, entry, "weapon_contract")
}
