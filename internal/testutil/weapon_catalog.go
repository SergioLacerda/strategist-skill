package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// CatalogProvider describes one plugins/catalog.yaml entry for a test fixture.
type CatalogProvider struct {
	ID            string
	Risk          string // risk_score
	CanonicalRole string // empty for a Weapon with no role affinity
	Source        string // compatibility_source; default "embedded"
	RuntimeKind   string // runtime.kind; default "host"
	NoPayload     bool   // skip writing skills/<id>/SKILL.md
}

// WriteWeaponCatalog writes a valid plugins/catalog.yaml under a .strategist root
// holding the given providers, and a skills/<id>/SKILL.md payload for each Weapon.
// It is the catalog-authority replacement for hand-writing skills/<id>/skill.yaml
// compat views in fixtures: a Weapon the catalog lists needs no view.
func WriteWeaponCatalog(t testing.TB, strategistRoot string, providers ...CatalogProvider) {
	t.Helper()
	var b strings.Builder
	b.WriteString("schema_version: strategist-plugin-catalog/v2\nproviders:\n")
	for _, p := range providers {
		writeCatalogEntry(&b, p)
		if p.NoPayload || p.Source == "native_role" {
			continue
		}
		dir := filepath.Join(strategistRoot, "skills", p.ID)
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# "+p.ID+"\n"), 0o644))
	}
	require.NoError(t, os.MkdirAll(filepath.Join(strategistRoot, "plugins"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(strategistRoot, "plugins", "catalog.yaml"), []byte(b.String()), 0o644))
}

// CatalogEntryYAML renders one catalog entry (two-space indented under providers:),
// for a test that extends a catalog it already wrote.
func CatalogEntryYAML(p CatalogProvider) string {
	var b strings.Builder
	writeCatalogEntry(&b, p)
	return b.String()
}

func writeCatalogEntry(b *strings.Builder, p CatalogProvider) {
	source, kind := p.Source, p.RuntimeKind
	if source == "" {
		source = "embedded"
	}
	if kind == "" {
		kind = "host"
	}
	b.WriteString("  - id: " + p.ID + "\n    risk_score: " + p.Risk + "\n    compatibility_source: " + source + "\n")
	if p.CanonicalRole != "" {
		b.WriteString("    canonical_role: " + p.CanonicalRole + "\n    roles:\n      - " + p.CanonicalRole + "\n")
		if p.CanonicalRole == "ranger" {
			b.WriteString("    weapon_contract:\n      role_owner: ranger\n      participation: required\n      invocation_evidence: required\n      unavailable_behavior: role_invocation_failed\n      native_substitution: forbidden\n")
		}
	}
	if source != "native_role" {
		b.WriteString("    runtime:\n      kind: " + kind + "\n")
	}
}
