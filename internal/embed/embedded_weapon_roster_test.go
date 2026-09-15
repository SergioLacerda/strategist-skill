package embed_test

import (
	"testing"

	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// embeddedWeaponRosterCatalogEntry is the minimal shape this test needs from
// a plugins/catalog.yaml provider entry — deliberately independent of
// internal/install's own unexported pluginCatalogProvider type, so this test
// exercises the actual embedded bytes a real build ships, not the ingestion
// pipeline's internal structs.
type embeddedWeaponRosterCatalogEntry struct {
	ID                  string `yaml:"id"`
	Installable         bool   `yaml:"installable"`
	CanonicalRole       string `yaml:"canonical_role"`
	CompatibilitySource string `yaml:"compatibility_source"`
}

type embeddedWeaponRosterCatalog struct {
	Providers []embeddedWeaponRosterCatalogEntry `yaml:"providers"`
}

// TestEmbeddedDefaults_BaselineWeaponRosterIsAlwaysEmbedded guards the
// requester's explicit item-3 baseline: brainstorming, openspec-explore, and
// openspec-propose live permanently under external-skills-source/ and, after
// any build, must be present in the compiled binary's embedded catalog as
// installable Wizard options (docs/adr/0035-embedded-weapon-fallback-policy.md
// DEC-001). This reads the actual embedded FS (internal/embed/defaults),
// i.e. exactly what ships in the binary — not external-skills-source/ or the
// ingestion pipeline directly (see TestEmbeddedSkillBaselineRoster* in
// internal/install for the source-side ingestion/drift guard).
func TestEmbeddedDefaults_BaselineWeaponRosterIsAlwaysEmbedded(t *testing.T) {
	t.Parallel()

	raw, err := embedpkg.Extractor{}.ReadFile("plugins/catalog.yaml")
	require.NoError(t, err)

	var catalog embeddedWeaponRosterCatalog
	require.NoError(t, yaml.Unmarshal(raw, &catalog))

	byID := make(map[string]embeddedWeaponRosterCatalogEntry, len(catalog.Providers))
	for _, p := range catalog.Providers {
		byID[p.ID] = p
	}

	for _, want := range []struct {
		id            string
		canonicalRole string
	}{
		{"brainstorming", "ranger"},
		{"openspec-explore", "ranger"},
		{"openspec-propose", "archivist"},
	} {
		entry, found := byID[want.id]
		require.Truef(t, found, "baseline weapon %q must always be present in the embedded catalog", want.id)
		assert.Equalf(t, "embedded", entry.CompatibilitySource, "%s must remain compatibility_source: embedded", want.id)
		assert.Truef(t, entry.Installable, "%s must remain installable (a real Wizard option)", want.id)
		assert.Equalf(t, want.canonicalRole, entry.CanonicalRole, "%s must declare canonical_role %q", want.id, want.canonicalRole)

		// The generated per-skill manifest mirror must also actually exist —
		// this is what a real strategist install extracts into a workspace's
		// .strategist/skills/<id>/skill.yaml.
		_, err := embedpkg.Extractor{}.ReadFile("skills/" + want.id + "/skill.yaml")
		require.NoErrorf(t, err, "skills/%s/skill.yaml must be embedded alongside its catalog entry", want.id)
	}
}
