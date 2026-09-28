package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const rosterCatalog = `schema_version: strategist-plugin-catalog/v2
providers:
  - id: brainstorming
    risk_score: write_analysis
    compatibility_source: embedded
    canonical_role: ranger
    roles: [ranger]
    weapon_contract:
      role_owner: ranger
      participation: required
      invocation_evidence: required
      unavailable_behavior: role_invocation_failed
      native_substitution: forbidden
    runtime:
      kind: embedded
  - id: openspec-propose
    risk_score: write_analysis
    compatibility_source: embedded
    canonical_role: archivist
    roles: [archivist]
    runtime:
      kind: openspec_root
      root: .strategist/openspec
  - id: openspec-archive-change
    risk_score: controlled
    compatibility_source: embedded
    canonical_role: sniper
    roles: [sniper]
    runtime:
      kind: host
  - id: sniper
    risk_score: controlled
    compatibility_source: native_role
  - id: openspec-apply-change
    risk_score: controlled
    compatibility_source: external
`

func rosterRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "plugins"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins", "catalog.yaml"), []byte(rosterCatalog), 0o644))
	writeDefaultRoleSlotMap(t, root)
	for role, slot := range map[string]string{"ranger": "discovery", "archivist": "refinement", "sniper": "execution"} {
		require.NoError(t, os.WriteFile(filepath.Join(root, "roles", role+".yaml"), []byte("role: "+role+"\nslot: "+slot+"\nextensibility: pluggable\n"), 0o644))
	}
	for _, id := range []string{"brainstorming", "openspec-propose", "openspec-archive-change"} {
		writePayload(t, root, id)
	}
	return root
}

// The roster comes from the catalog (compatibility_source embedded plus a
// canonical_role): with no compat view anywhere, every embedded Weapon is verified,
// and an unbound one (openspec-archive-change) is verified, not reported missing.
func TestRosterComesFromTheCatalogWithoutAnyCompatView(t *testing.T) {
	root := rosterRoot(t)

	bindings, err := verifyEmbeddedWeaponBindings(root)

	require.NoError(t, err)
	for _, id := range []string{"brainstorming", "openspec-propose", "openspec-archive-change"} {
		b, ok := findBinding(bindings, id)
		require.True(t, ok, id)
		assert.Truef(t, b.OK, "%s: %s", id, b.Reason)
	}
	_, native := findBinding(bindings, "sniper")
	assert.False(t, native, "a native_role entry is not a Weapon")
	_, external := findBinding(bindings, "openspec-apply-change")
	assert.False(t, external, "an external entry without a canonical_role is not part of the embedded roster")
	assert.Len(t, bindings, 3)
}

func TestRosterReportsAnEmbeddedWeaponWhoseSkillPayloadIsMissing(t *testing.T) {
	root := rosterRoot(t)
	require.NoError(t, os.Remove(filepath.Join(root, "skills", "brainstorming", "SKILL.md")))

	bindings, err := verifyEmbeddedWeaponBindings(root)

	require.NoError(t, err)
	b, ok := findBinding(bindings, "brainstorming")
	require.True(t, ok)
	assert.False(t, b.OK)
	assert.Contains(t, b.Reason, "SKILL.md")
}

func TestRosterDoesNotDuplicateACatalogedWeaponThatAlsoHasAView(t *testing.T) {
	root := rosterRoot(t)
	writeWeaponFixture(t, root, "brainstorming", "ranger", "")

	bindings, err := verifyEmbeddedWeaponBindings(root)

	require.NoError(t, err)
	count := 0
	for _, b := range bindings {
		if b.SkillID == "brainstorming" {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

// A hand-placed view for a Weapon the catalog does not list is still scanned
// (transitional, DEC-013).
func TestRosterStillScansAnUncatalogedViewDuringTheTransition(t *testing.T) {
	root := rosterRoot(t)
	writeWeaponFixture(t, root, "hand-made", "archivist", "")

	bindings, err := verifyEmbeddedWeaponBindings(root)

	require.NoError(t, err)
	_, ok := findBinding(bindings, "hand-made")
	assert.True(t, ok)
}
